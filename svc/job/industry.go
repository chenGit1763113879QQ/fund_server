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

	var industries []struct {
		IndCode    string `json:"encode" bson:"_id"`
		Symbol     string
		MarketType uint8  `bson:"marketType"`
		Type       uint8  `bson:"type"`
		Name       string `json:"name"`
	}
	util.UnmarshalJSON(body, &industries, "data", "industries")

	// stocks
	url = fmt.Sprintf("https://xueqiu.com/service/screener/screen?category=%s&areacode=&indcode=&size=6000&only_count=0", market)
	body, _ = util.GetAndRead(url)

	var stock []struct {
		Code    string `json:"symbol"`
		IndCode string `json:"indcode"`
	}
	util.UnmarshalJSON(body, &stock, "data", "list")

	// save
	bulk := db.Stock.Bulk()

	var prefix string

	for _, ids := range industries {
		// type
		if ids.IndCode[0:3] != prefix {
			ids.Type = util.TYPE_I1
			prefix = ids.IndCode[0:3]
		} else {
			ids.Type = util.TYPE_I2
		}

		for _, stk := range stock {
			if ids.IndCode == stk.IndCode {

				switch market {
				case "CN":
					ids.MarketType = util.MARKET_CN
					stk.Code = fmt.Sprintf("%s.%s", stk.Code[2:], stk.Code[0:2])

				case "HK":
					ids.MarketType = util.MARKET_HK
					stk.Code += ".HK"

				case "US":
					ids.MarketType = util.MARKET_US
					stk.Code += ".US"
				}
				ids.Symbol = ids.IndCode

				db.Stock.InsertOne(ctx, ids)

				// industry
				bulk.UpdateId(ids.IndCode, bson.M{
					"$set": bson.M{
						"name": ids.Name, "marketType": ids.MarketType, "type": ids.Type,
					},
					"$addToSet": bson.M{"members": stk.Code},
				})
				// member
				bulk.UpdateId(stk.Code, bson.M{"$addToSet": bson.M{"bk": ids.IndCode}})
			}
		}
	}
	bulk.Run(ctx)
	log.Info().Msgf("init industry[%s] success", market)
}

func getIndustry(m *model.Market) {
	var data []model.Industry

	db.Stock.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"type": bson.M{"$in": []int{util.TYPE_I1, util.TYPE_I2}}}).
		Lookup("stock", "members", "_id", "c").
		Project(bson.M{
			"c":         bson.M{"name": 1, "pct_chg": 1},
			"name":      "$name",
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

	pinyinArg := pinyin.NewArgs()

	for _, i := range data {
		i.PctLeader.PctChg = -100

		// leader stock
		for _, stk := range i.ConnList {
			if stk.PctChg > i.PctLeader.PctChg {
				i.PctLeader = stk
			}
		}
		i.ConnList = nil

		if m.Freq() == 2 {
			// add pinyin
			if util.IsChinese(i.Name) {
				for _, c := range pinyin.LazyPinyin(i.Name, pinyinArg) {
					i.Pinyin += c
					i.LazyPinyin += string(c[0])
				}
			}
		}

		bulk.UpdateId(i.Id, bson.M{"$set": i})

		// minute data
		if m.Status {
			minBulk.UpsertId(
				bson.M{"code": i.Id, "time": newTime.Unix()},
				bson.M{"pct_chg": i.PctChg, "vol": i.Vol, "main_net": i.MainNet},
			)
		}
	}
	bulk.Run(ctx)
	minBulk.Run(ctx)
}
