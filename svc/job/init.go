package job

import (
	"context"
	"fund/model"
	"fund/util"
	"sync"
	"time"

	"github.com/xgzlucario/structx"
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
	structx.GoJob(getMarketStatus, time.Second)

	// stock
	for _, p := range Markets {
		go getRealStock(p)
	}
	time.Sleep(time.Second * 3)

	// industry
	structx.GoJob(func() {
		for _, p := range Markets {
			go getIndustries(p)
		}
	}, time.Hour*24)

	// kline & predict
	// structx.GoJob(func() {
	// 	initKline()
	// 	WinRate()
	// }, time.Hour*24)
}
