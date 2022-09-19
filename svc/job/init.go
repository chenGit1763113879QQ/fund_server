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

	// news
	util.GoJob(getNews, time.Minute, time.Second*3)

	// market status
	util.GoJob(getMarketStatus, time.Second)

	// industry
	util.GoJob(func() {
		for _, p := range Markets {
			getCategoryIndustries(p.StrMarket)
			getIndustry(p)
		}
	}, time.Hour*12, time.Second*5)

	// kline & predict
	util.GoJob(func() {
		InitKlines()

		// loadKlines()

		// WinRate()
		// PredictStock()

		// cache.KlineMap.Clear()

	}, time.Hour*12, time.Second*5)
}
