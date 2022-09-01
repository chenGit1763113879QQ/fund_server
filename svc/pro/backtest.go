package pro

import (
	"fund/db"
	"fund/model"
	"fund/util"

	"go.mongodb.org/mongo-driver/bson"
)

const (
	SIDE_BUY uint8 = iota
	SIDE_SELL
)

func test1(arg float64) {
	title := "test1"

	trade := model.NewTrade(title, arg)
	coll := db.BackDB.Collection(title)

	klineMap.Range(func(id string, k []model.Kline) {
		for i := range k {
			if k[i].WinnerRate < 2.7 && k[i].Tr < 3.5 && k[i].Pe < 33 {
				trade.Buy(k[i])
				coll.UpsertId(ctx,
					bson.M{"code": id, "time": k[i].Time},
					bson.M{"type": SIDE_BUY, "close": k[i].Close, "winner_rate": k[i].WinnerRate})

			} else if k[i].WinnerRate > arg {
				trade.Sell(k[i])
				coll.UpsertId(ctx,
					bson.M{"code": id, "time": k[i].Time},
					bson.M{"type": SIDE_SELL, "close": k[i].Close, "winner_rate": k[i].WinnerRate, "profit": trade.Profit})
			}
		}
	})
}

func Test1() {
	p := util.NewPool(5)
	p.NewTask(func() { test1(20) })
	p.NewTask(func() { test1(30) })
	p.NewTask(func() { test1(40) })
	p.NewTask(func() { test1(50) })
	p.NewTask(func() { test1(60) })
	p.Wait()
}
