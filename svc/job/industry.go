package job

import (
	"fmt"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

func getCategoryIndustries(m *model.Market) {
	// industry
	url := fmt.Sprintf("https://xueqiu.com/service/screener/industries?category=%s", m.StrMarket)
	body, _ := util.GetAndRead(url)

	var idsData []*model.Industry
	util.UnmarshalJSON(body, &idsData, "data", "industries")

	// stock
	url = fmt.Sprintf("https://xueqiu.com/service/screener/screen?category=%s&areacode=&indcode=&size=6000&only_count=0", m.StrMarket)
	body, _ = util.GetAndRead(url)

	var stock []*struct {
		Code    string `json:"symbol"`
		IndCode string `json:"indcode"`
	}
	util.UnmarshalJSON(body, &stock, "data", "list")
	for _, s := range stock {
		if len(s.Code) >= 2 {
			s.Code = fmt.Sprintf("%s.%s", s.Code[2:], s.Code[0:2])
		}
	}

	// save
	bulk := db.Stock.Bulk()

	for _, ids := range idsData {
		// set basic
		ids.Id = ids.Symbol
		ids.MarketType = m.Market
		ids.Type = util.TYPE_IDS
		ids.AddPinYin(ids.Name)

		// save
		bulk.UpsertId(ids.Id, ids)

		// members
		for _, stk := range stock {
			if ids.Id == stk.IndCode {
				// members
				bulk.UpdateId(ids.Id, bson.M{"$addToSet": bson.M{"members": stk.Code}})
				// bk
				bulk.UpdateId(stk.Code, bson.M{"$addToSet": bson.M{"bk": ids.Id}})
			}
		}
	}
	bulk.Run(ctx)
	log.Info().Msgf("init industry[%s] success", m.StrMarket)
}

func getIndustry(m *model.Market) {
	var data []*model.Industry

	db.Stock.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"marketType": m.Market, "type": util.TYPE_IDS}).
		Lookup("stock", "members", "_id", "c").
		Project(bson.M{
			"symbol":    1,
			"followers": bson.M{"$sum": "$c.followers"},
			"pct_chg":   bson.M{"$avg": "$c.pct_chg"},
			"main_net":  bson.M{"$sum": "$c.main_net"},
			"vol":       bson.M{"$sum": "$c.vol"},
			"tr":        bson.M{"$avg": "$c.tr"},
			"amount":    bson.M{"$sum": "$c.amount"},
			"mc":        bson.M{"$sum": "$c.mc"},
			"fmc":       bson.M{"$sum": "$c.fmc"},
			"pe_ttm":    bson.M{"$avg": "$c.pe_ttm"},
			"pb":        bson.M{"$avg": "$c.pb"},
			"pct_year":  bson.M{"$avg": "$c.pct_year"},
			"count":     bson.M{"$size": "$c"},
		}).Do()).All(&data)

	bulk := db.Stock.Bulk()

	// tradeTime
	tradeTime := m.TradeTime.Format("2006/01/02 15:04")
	date := strings.Split(tradeTime, " ")[0]

	newTime, _ := time.Parse("2006/01/02 15:04", tradeTime)

	minBulk := db.MinuteDB.Collection(date).Bulk()

	for _, i := range data {
		i.Id = i.Symbol
		bulk.UpdateId(i.Id, bson.M{"$set": i})

		// minute data
		if m.Status {
			id := fmt.Sprintf("%s-%s", i.Id, tradeTime)
			minBulk.UpsertId(
				id,
				bson.M{"code": i.Id, "time": newTime.Unix(),
					"pct_chg": i.PctChg, "vol": i.Vol,
					"main_net": i.MainNet, "minute": newTime.Minute()},
			)
		}
	}
	bulk.Run(ctx)
	minBulk.Run(ctx)
}
