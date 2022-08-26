package pro

import (
	"context"
	"fund/cache"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"math"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

type KlineMap struct {
	data map[string][]model.Kline
	sync.RWMutex
}

var (
	ctx      = context.Background()
	klineMap = &KlineMap{data: make(map[string][]model.Kline)}
)

func (s *KlineMap) Length() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.data)
}

func (s *KlineMap) Load(key string) []model.Kline {
	s.RLock()
	defer s.RUnlock()
	return s.data[key]
}

func (s *KlineMap) Store(key string, value []model.Kline) {
	s.Lock()
	defer s.Unlock()
	s.data[key] = value
}

func (s *KlineMap) Range(f func(k string, v []model.Kline)) {
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.data {
		f(k, v)
	}
}

func initKline() {
	p := util.NewPool(5)
	cache.Stock.RangeForCNStock(func(k string, v model.Stock) {
		// filter
		if v.Mc > 50*math.Pow(10, 8) {
			p.NewTask(func() {
				var data []model.Kline
				// get kline
				db.KlineDB.Collection(util.Md5Code(k)).Aggregate(ctx, mongox.Pipeline().
					Match(bson.M{"code": k}).Sort(bson.M{"time": 1}).Do()).All(&data)
				klineMap.Store(k, data)
			})
		}
	})
	p.Wait()
	log.Info().Msgf("init kline success, length: %d", klineMap.Length())
}

func Init() {
	// wait for cache init
	time.Sleep(time.Second * 5)
	initKline()
	for {
		log.Info().Msg("start jobs...")
		Test1()
		log.Info().Msg("jobs finished")
		time.Sleep(time.Hour)
	}
}
