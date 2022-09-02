package pro

import (
	"fund/db"
	"fund/model"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	TYPE_WINRATE = "win_rate"
)

const (
	ARG_WINRATE = "winner_rate"
)

func Init() {
	// wait for init
	time.Sleep(time.Second * 5)
	initKline()
	
	for {
		log.Debug().Msg("start jobs...")
		WinRate()
		log.Debug().Msg("jobs finished")
		time.Sleep(time.Hour)
	}
}

// run back test
func runBackTest(backType string, arg float64, argName string, buy func(k model.Kline) bool, sell func(k model.Kline) bool) {
	// init collection
	coll := db.BackDB.Collection(backType)
	coll.EnsureIndexes(ctx, []string{"code,arg"}, nil)

	coll.Remove(ctx, bson.M{"arg": arg, "arg_name": argName})
	bulk := coll.Bulk()

	klineMap.Range(func(id string, k []model.Kline) {
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
