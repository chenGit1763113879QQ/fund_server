package pro

import (
	"fmt"
	"fund/model"
)

func init() {
	initKline()
}

// total 24431 profit 0.203
func test1(arg float64) {
	trade := model.NewTrade(fmt.Sprintf("test arg:%.2f", arg))

	KlineMap.Range(func(key, value any) bool {
		trade.Init()
		id := key.(string)
		k := value.([]model.Kline)

		for i := range k {
			if k[i].WinnerRate < 2.7 && k[i].Tr < 3.5 && k[i].Pe < 33 {
				trade.Buy(k[i])

			} else if k[i].WinnerRate > arg {
				trade.Sell(k[i], id)
			}
		}
		return true
	})
	trade.RecordsInfo()
}

// untest
func test2(arg float64) {
	trade := model.NewTrade(fmt.Sprintf("test arg:%.2f", arg))

	KlineMap.Range(func(key, value any) bool {
		trade.Init()
		id := key.(string)
		k := value.([]model.Kline)

		for i := range k {
			if k[i].KDJ_J < 20 && k[i].Tr < 3.5 && k[i].Pe < 33 {
				trade.Buy(k[i])

			} else if k[i].KDJ_J > arg {
				trade.Sell(k[i], id)
			}
		}
		return true
	})
	trade.RecordsInfo()
}

func Test() {
	go test2(70)
	go test2(80)
	go test2(75)
	go test2(85)
	go test2(90)
}
