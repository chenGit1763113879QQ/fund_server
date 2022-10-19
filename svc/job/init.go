package job

import (
	"context"
	"fund/model"
	"fund/util"
	"sync"
	"time"
)

var (
	ctx = context.Background()

	Markets = []*model.Market{
		{Market: util.CN, Type: util.STOCK, StrMarket: "CN", StrType: "sh_sz"},
		{Market: util.HK, Type: util.STOCK, StrMarket: "HK", StrType: "hk"},
		{Market: util.US, Type: util.STOCK, StrMarket: "US", StrType: "us"},
	}

	Cond = sync.NewCond(&sync.Mutex{})
)

func init() {
	getMarketStatus()
	for _, p := range Markets {
		go getRealStock(p)
	}

	// market status
	util.GoJob(getMarketStatus, time.Second)

	// industry
	util.GoJob(func() {
		for _, p := range Markets {
			getCategoryIndustries(p)
			go getIndustry(p)
		}
	}, time.Hour*24, time.Second*3)

	// kline & predict
	util.GoJob(func() {
		initKline()
		loadKline()

		WinRate()
		PredictStock()
	}, time.Hour*24, time.Second*3)
}
