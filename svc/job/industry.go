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

func getIndustries(m *model.Market) {
	// industry
	url := fmt.Sprintf("https://xueqiu.com/service/screener/industries?category=%v", m.Market)
	body, err := util.GetAndRead(url)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	var industries []*model.Industry
	util.UnmarshalJSON(body, &industries, "data", "industries")

	for _, item := range industries {
		// industry set basic
		item.Id = item.Symbol
		item.MarketType = m.Market
		item.Type = util.IDS
		item.AddPinYin(item.Name)

		// stock
		url = fmt.Sprintf("https://xueqiu.com/service/screener/screen?category=%v&indcode=%v&size=500&only_count=0", m.Market, item.Symbol)
		body, err = util.GetAndRead(url)
		if err != nil {
			log.Error().Msg(err.Error())
		}

		var stock []struct {
			Code string `json:"symbol"`
		}
		util.UnmarshalJSON(body, &stock, "data", "list")

		item.Members = make([]string, len(stock))

		bulk := db.Stock.Bulk()
		for i, s := range stock {
			s.Code = util.ParseCode(s.Code)
			// stock bk
			bulk.UpdateId(s.Code, bson.M{"$set": bson.M{"bk": item.Id}})

			item.Members[i] = s.Code
		}
		// save
		bulk.InsertOne(item).UpdateId(item.Id, bson.M{"$set": item}).Run(ctx)

		time.Sleep(time.Second / 10)
	}

	log.Info().Msgf("init industry[%s] success", m.Market)
}

func getIndustry(m *model.Market) {
	var data []*model.Industry

	err := db.Stock.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"marketType": m.Market, "type": util.IDS}).
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

	if err != nil {
		log.Error().Msg(err.Error())
	}

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
