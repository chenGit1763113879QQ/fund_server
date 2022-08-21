package pro

import (
	"fmt"
	"fund/model"
)

func test1(arg float64) {
	trade := model.NewTrade(fmt.Sprintf("test arg:%.2f", arg))

	klineMap.Range(func(id string, k []model.Kline) {
		trade.Init()

		for i := range k {
			if k[i].WinnerRate < 2.7 && k[i].Tr < 3.5 && k[i].Pe < 33 {
				trade.Buy(k[i])

			} else if k[i].WinnerRate > arg {
				trade.Sell(k[i], id)
			}
		}
	})
	trade.RecordsInfo()
}

// untest
func test2(arg float64) {
	trade := model.NewTrade(fmt.Sprintf("test arg:%.2f", arg))

	klineMap.Range(func(id string, k []model.Kline) {
		trade.Init()

		for i := range k {
			if k[i].KDJ_J < 20 && k[i].Tr < 3.5 && k[i].Pe < 33 {
				trade.Buy(k[i])

			} else if k[i].KDJ_J > arg {
				trade.Sell(k[i], id)
			}
		}
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
