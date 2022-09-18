package pro

import (
	"context"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
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
	log.Debug().Msg("kline start init")
	t, _ := time.Parse("2006/01/02", "2017/01/01")

	p := util.NewPool()
	for _, code := range getCNStocks() {
		p.NewTask(func(strs ...string) {
			id := strs[0]
			var data []model.Kline

			// get kline
			db.KlineDB.Collection(util.Md5Code(id)).Aggregate(ctx, mongox.Pipeline().
				Match(bson.M{"code": id, "time": bson.M{"$gt": t}}).
				Sort(bson.M{"time": 1}).Do()).All(&data)
			if data != nil {
				klineMap.Store(id, data)
			}
		}, code)
	}
	p.Wait()

	log.Debug().Msgf("init kline success, length:%d", len(klineMap.data))
}
