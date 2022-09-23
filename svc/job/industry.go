package job

import (
	"fmt"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"strings"
	"time"

	"github.com/mozillazg/go-pinyin"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

func getCategoryIndustries(market string) {
	// industries
	url := fmt.Sprintf("https://xueqiu.com/service/screener/industries?category=%s", market)
	body, _ := util.GetAndRead(url)

	var industries []*struct {
		MarketType util.Code `bson:"marketType"`
		Type       util.Code `bson:"type"`
		Code       string    `json:"encode" bson:"_id"`
		Symbol     string
		Name       string `json:"name"`
		Pinyin     string `bson:"pinyin"`
		LazyPinyin string `bson:"lazy_pinyin"`
	}
	util.UnmarshalJSON(body, &industries, "data", "industries")

	// stocks
	url = fmt.Sprintf("https://xueqiu.com/service/screener/screen?category=%s&areacode=&indcode=&size=6000&only_count=0", market)
	body, _ = util.GetAndRead(url)

	var stock []*struct {
		Code    string `json:"symbol"`
		IndCode string `json:"indcode"`
	}
	util.UnmarshalJSON(body, &stock, "data", "list")

	// save
	bulk := db.Stock.Bulk()

	for _, ids := range industries {
		// set industry
		ids.Type = util.TYPE_IDS
		ids.Symbol = ids.Code
		switch market {
		case "CN":
			ids.MarketType = util.MARKET_CN

		case "HK":
			ids.MarketType = util.MARKET_HK

		case "US":
			ids.MarketType = util.MARKET_US
		}

		// add pinyin
		if util.IsChinese(ids.Name) {
			for _, c := range pinyin.LazyPinyin(ids.Name, model.PinyinArg) {
				ids.Pinyin += c
				ids.LazyPinyin += string(c[0])
			}
		}

		// save
		db.Stock.InsertOne(ctx, ids)
		bulk.UpdateId(ids.Code, bson.M{"$set": ids})

		// members
		for _, stk := range stock {
			if ids.Code == stk.IndCode {
				switch market {
				case "CN":
					stk.Code = fmt.Sprintf("%s.%s", stk.Code[2:], stk.Code[0:2])

				case "HK":
					stk.Code += ".HK"

				case "US":
					stk.Code += ".US"
				}

				// members
				bulk.UpdateId(ids.Code, bson.M{"$addToSet": bson.M{"members": stk.Code}})
				// bk
				bulk.UpdateId(stk.Code, bson.M{"$addToSet": bson.M{"bk": ids.Code}})
			}
		}
	}
	bulk.Run(ctx)
	log.Info().Msgf("init industry[%s] success", market)
}

func getIndustry(m *model.Market) {
	var data []*model.Industry

	db.Stock.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"marketType": m.Market, "type": util.TYPE_IDS}).
		Lookup("stock", "members", "_id", "c").
		Project(bson.M{
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
		}).Do()).All(&data)

	bulk := db.Stock.Bulk()

	// tradeTime
	tradeTime := m.TradeTime.Format("2006/01/02 15:04")
	date := strings.Split(tradeTime, " ")[0]

	newTime, _ := time.Parse("2006/01/02 15:04", tradeTime)

	minBulk := db.MinuteDB.Collection(date).Bulk()

	for _, i := range data {
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
