package pro

import (
	"fund/db"
	"fund/model"
)

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
}
