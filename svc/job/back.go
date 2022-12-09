package job

import (
	"fund/db"
	"fund/model"

	"github.com/rs/zerolog/log"
)

const (
	ALG_WINRATE = "winner_rate"
)

type backFunc func(k *model.Kline) bool

// run back test
func runBackTest(backType string, arg float64, argName string, buy backFunc, sell backFunc) {
	// // init collection
	// coll := db.BackDB.Collection(backType)
	// coll.EnsureIndexes(ctx, []string{"code,arg"}, nil)

	// bulk := coll.Bulk()

	// // run
	// cache.RangeKline(func(id string, klines []*model.Kline) {
	// 	trade := model.NewTrade(id, arg, argName)
	// 	for _, k := range klines {
	// 		if buy(k) {
	// 			trade.Buy(k)

	// 		} else if sell(k) {
	// 			trade.Sell(k)
	// 		}
	// 	}
	// 	bulk.InsertOne(trade)
	// })
	// bulk.Run(ctx)
}

func WinRate() {
	db.BackDB.Collection(ALG_WINRATE).DropCollection(ctx)

	runBackTest(
		ALG_WINRATE, 10.0, ALG_WINRATE,
		func(k *model.Kline) bool { return k.WinnerRate < 2.7 && k.Tr < 3.5 && k.Pe < 33 },
		func(k *model.Kline) bool { return k.WinnerRate > 20 },
	)
	log.Debug().Msgf("%s backtest finished", ALG_WINRATE)
}
