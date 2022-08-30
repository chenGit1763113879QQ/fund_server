package pro

import (
	"fund/db"
	"fund/model"
	"fund/util"

	"go.mongodb.org/mongo-driver/bson"
)

// 策略1：低买高卖
func test1(arg float64) {
	klineMap.Range(func(id string, k []model.Kline) {
		for i := range k {
			if k[i].WinnerRate < 2.7 && k[i].Tr < 3.5 && k[i].Pe < 33 {
				_id := bson.M{"code": id, "time": k[i].Time}

				db.Backtest.UpsertId(ctx,
					_id, bson.M{"type": "buy", "close": k[i].Close, "arg": arg, "winner_rate": k[i].WinnerRate})

			} else if k[i].WinnerRate > arg {
				_id := bson.M{"code": id, "time": k[i].Time}

				db.Backtest.UpsertId(ctx,
					_id, bson.M{"type": "sell", "close": k[i].Close, "arg": arg, "winner_rate": k[i].WinnerRate})
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
