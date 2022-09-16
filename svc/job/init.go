package job

import (
	"context"
	"fund/model"
	"fund/util"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const XQHOST = "https://xueqiu.com/service/v5/stock"

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
	log.Info().Msg("init market status success.")

	for _, p := range Markets {
		go getRealStock(p)
	}

	util.GoJob(getNews, time.Minute, time.Second*3)

	util.GoJob(getMarketStatus, time.Second)

	util.GoJob(func() {
		// industry
		for _, p := range Markets {
			getCategoryIndustries(p.StrMarket)
		}
		// kline
		InitKlines()
	}, time.Hour*12, time.Second*5)
}
