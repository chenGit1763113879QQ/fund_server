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
		{Market: util.CN, Type: util.STOCK},
		{Market: util.HK, Type: util.STOCK},
		{Market: util.US, Type: util.STOCK},
	}

	Cond = sync.NewCond(&sync.Mutex{})
)

func init() {
	getMarketStatus()
	// market status
	util.GoJob(getMarketStatus, time.Second)

	// stock
	for _, p := range Markets {
		go getRealStock(p)
	}
	time.Sleep(time.Second * 3)

	// industry
	util.GoJob(func() {
		for _, p := range Markets {
			go getIndustries(p)
		}
	}, time.Hour*24)

	// kline
	util.GoJob(initKline, time.Hour*24)
}
