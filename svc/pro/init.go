package pro

import (
	"context"
	"fund/cache"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"fund/util/pool"
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

var (
	ctx      = context.Background()
	klineMap = &KlineMap{data: make(map[string][]model.Kline)}
)

func initKline() {
	p := pool.NewPool(5)
	cache.Stock.RangeForCNStock(func(k string, v model.Stock) {
		// filter
		if v.Mc > 50*math.Pow(10, 8) {
			p.NewTask(func() {
				klineMap.Store(k, getKline(k))
			})
		}
	})
	p.Wait()
	log.Info().Msgf("init kline success, length: %d", klineMap.Length())
}

func getKline(code string) []model.Kline {
	var data []model.Kline
	db.KlineDB.Collection(util.CodeToInt(code)).Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"code": code}).Sort(bson.M{"time": 1}).Do()).All(&data)
	return data
}

func Init() {
	// 等待缓存初始化
	time.Sleep(time.Second * 5)
	initKline()
	for {
		log.Info().Msg("start jobs...")
		Test1()
		time.Sleep(time.Hour)
	}
}
