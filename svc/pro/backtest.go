package pro

import (
	"fmt"
	"fund/db"
	"fund/model"

	"go.mongodb.org/mongo-driver/bson"
)

func test1(arg float64) {
	trade := model.NewTrade(fmt.Sprintf("test arg:%.2f", arg))

	klineMap.Range(func(id string, k []model.Kline) {
		trade.Init()

		for i := range k {
			if k[i].WinnerRate < 2.7 && k[i].Tr < 3.5 && k[i].Pe < 33 {
				// log
				_id := bson.M{"code": id, "time": k[i].Time}
				db.Backtest.UpsertId(ctx,
					_id,
					bson.M{"type": "b", "close": k[i].Close, "arg": arg, "winner_rate": k[i].WinnerRate})

				trade.Buy(k[i])

			} else if k[i].WinnerRate > arg {
				// log
				_id := bson.M{"code": id, "time": k[i].Time}
				db.Backtest.UpsertId(ctx,
					_id,
					bson.M{"type": "s", "close": k[i].Close, "arg": arg, "winner_rate": k[i].WinnerRate})

				trade.Sell(k[i], id)
			}
		}
	})
	trade.RecordsInfo()
}

func Test() {
	go test1(70)
	go test1(80)
	go test1(75)
	go test1(85)
	go test1(90)
}
