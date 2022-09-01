package pro

import (
	"fund/db"
	"fund/model"
)

func test1(arg float64) {
	bulk := db.BackDB.Collection("test1").Bulk()

	klineMap.Range(func(id string, k []model.Kline) {
		trade := model.NewTrade(id, arg, "winner_rate")

		for i := range k {
			if k[i].WinnerRate < 2.7 && k[i].Tr < 3.5 && k[i].Pe < 33 {
				trade.Buy(k[i])

			} else if k[i].WinnerRate > arg {
				trade.Sell(k[i])
			}
		}
		bulk.InsertOne(trade)
	})
	bulk.Run(ctx)
}

func Test1() {
	// drop collection
	db.BackDB.Collection("test1").DropCollection(ctx)

	for i := 1; i < 6; i++ {
		test1(float64(i * 10))
	}
}
