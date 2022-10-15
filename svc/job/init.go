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
		{Market: util.MARKET_CN, Type: util.TYPE_STOCK, StrMarket: "CN", StrType: "sh_sz"},
		{Market: util.MARKET_HK, Type: util.TYPE_STOCK, StrMarket: "HK", StrType: "hk"},
		{Market: util.MARKET_US, Type: util.TYPE_STOCK, StrMarket: "US", StrType: "us"},
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
			getIndustry(p)
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
