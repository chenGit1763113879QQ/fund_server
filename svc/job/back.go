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

type BACK_FUNC func(k model.Kline) bool

// run back test
func runBackTest(backType string, arg float64, argName string, buy BACK_FUNC, sell BACK_FUNC) {
	// init collection
	coll := db.BackDB.Collection(backType)
	coll.EnsureIndexes(ctx, []string{"code,arg"}, nil)

	coll.Remove(ctx, bson.M{"arg": arg, "arg_name": argName})
	bulk := coll.Bulk()

	cache.KlineMap.Range(func(id string, k []model.Kline) {
		trade := model.NewTrade(id, arg, argName)
		for i := range k {
			if buy(k[i]) {
				trade.Buy(k[i])

			} else if sell(k[i]) {
				trade.Sell(k[i])
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
			func(k model.Kline) bool {
				return k.WinnerRate < 2.7 && k.Tr < 3.5 && k.Pe < 33
			},
			func(k model.Kline) bool {
				return k.WinnerRate > arg
			},
		)
	}
	log.Debug().Msgf("%s backtest finished", TYPE_WINRATE)
}
