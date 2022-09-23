package job

import (
	"fund/cache"
	"fund/db"
	"fund/model"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	TYPE_WINRATE = "win_rate"

	ARG_WINRATE = "winner_rate"
)

type BACK_FUNC func(k *model.Kline) bool

// run back test
func runBackTest(backType string, arg float64, argName string, buy BACK_FUNC, sell BACK_FUNC) {
	// init collection
	coll := db.BackDB.Collection(backType)
	coll.EnsureIndexes(ctx, []string{"code,arg"}, nil)

	// rm collection
	coll.Remove(ctx, bson.M{"arg": arg, "arg_name": argName})
	bulk := coll.Bulk()

	// run
	cache.KlineMap.Range(func(id string, klines []*model.Kline) {
		trade := model.NewTrade(id, arg, argName)
		for _, k := range klines {
			if buy(k) {
				trade.Buy(k)

			} else if sell(k) {
				trade.Sell(k)
			}
		}
		bulk.InsertOne(trade)
	})
	bulk.Run(ctx)
}

func WinRate() {
	db.BackDB.Collection(TYPE_WINRATE).DropCollection(ctx)

	for i := 1; i < 6; i++ {
		arg := float64(i * 10)

		runBackTest(
			TYPE_WINRATE, arg, ARG_WINRATE,
			func(k *model.Kline) bool {
				return k.WinnerRate < 2.7 && k.Tr < 3.5 && k.Pe < 33
			},
			func(k *model.Kline) bool {
				return k.WinnerRate > arg
			},
		)
	}
	log.Debug().Msgf("%s backtest finished", TYPE_WINRATE)
}
