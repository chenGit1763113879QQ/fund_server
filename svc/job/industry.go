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
		time.Sleep(time.Second * 5)
		for _, p := range Markets {
			getCategoryIndustries(p.StrMarket)
		}
	}()
}

func getCategoryIndustries(market string) {
	// industries
	var industries []struct {
		IndCode    string `json:"encode" bson:"_id"`
		MarketType uint8  `bson:"marketType"`
		Type       uint8  `bson:"type"`
		Name       string `json:"name"`
	}

	url := fmt.Sprintf("%s/industries?category=%s", XUEQIU, market)
	body, _ := util.GetAndRead(url)

	util.UnmarshalJSON(body, &industries, "data", "industries")

	// stock
	var stock []struct {
		Code    string `json:"symbol"`
		IndCode string `json:"indcode"`
	}

	url = fmt.Sprintf("%s/screen?category=%s&areacode=&indcode=&size=6000&only_count=0", XUEQIU, market)
	body, _ = util.GetAndRead(url)

	util.UnmarshalJSON(body, &stock, "data", "list")

	// save
	bulk := db.Stock.Bulk()

	for _, ids := range industries {
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
				ids.Type = util.TYPE_IDS

				db.Stock.InsertOne(ctx, ids)

				bulk.UpdateId(ids.IndCode, bson.M{
					"$set": bson.M{
						"name": ids.Name, "marketType": ids.MarketType, "type": ids.Type,
					},
					"$addToSet": bson.M{"members": stk.Code},
				})
				bulk.UpdateId(stk.Code, bson.M{"$addToSet": bson.M{"bk": ids.IndCode}})
			}
		}
	}
	bulk.Run(ctx)
	log.Info().Msgf("init industry[%s] success", market)
}
