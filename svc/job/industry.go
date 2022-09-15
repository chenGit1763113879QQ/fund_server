package job

import (
	"fmt"
	"fund/db"
	"fund/util"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

const XUEQIU = "https://xueqiu.com/service/screener"

func init() {
	go func() {
		for {
			time.Sleep(time.Second * 5)
			for _, p := range Markets {
				getCategoryIndustries(p.StrMarket)
			}
			time.Sleep(time.Hour)
		}
	}()
}

func getCategoryIndustries(market string) {
	// industries
	url := fmt.Sprintf("%s/industries?category=%s", XUEQIU, market)
	body, _ := util.GetAndRead(url)

	var industries []struct {
		IndCode    string `json:"encode" bson:"_id"`
		MarketType uint8  `bson:"marketType"`
		Type       uint8  `bson:"type"`
		Name       string `json:"name"`
	}
	util.UnmarshalJSON(body, &industries, "data", "industries")

	// stocks
	url = fmt.Sprintf("%s/screen?category=%s&areacode=&indcode=&size=6000&only_count=0", XUEQIU, market)
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
