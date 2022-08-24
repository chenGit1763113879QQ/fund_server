package pro

import (
	"context"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"fund/util/pool"
	"math"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

type Stockx struct {
	Id     string `bson:"_id"`
	Market string `bson:"marketType"`
	Name   string
	Mc     float64
}

type KlineMap struct {
	data map[string][]model.Kline
	mu   sync.RWMutex
}

func (s *KlineMap) Load(key string) []model.Kline {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[key]
}

func (s *KlineMap) Store(key string, value []model.Kline) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *KlineMap) Range(f func(k string, v []model.Kline)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.data {
		f(k, v)
	}
}

var (
	ctx      = context.Background()
	items    []Stockx
	klineMap = &KlineMap{data: make(map[string][]model.Kline)}
)

func init() {
	initStock()
	initKline()
}

func initStock() {
	db.Stock.Find(ctx, bson.M{
		"type": "stock", "marketType": "CN", "mc": bson.M{"$gt": 50 * math.Pow(10, 8)},
	}).Sort("-amount").All(&items)

	log.Info().Msgf("init stock[%d] success", len(items))
}

func initKline() {
	p := pool.NewPool(8)
	for _, i := range items {
		p.NewTask(func() {
			klineMap.Store(i.Id, i.getKline())
		})
	}
	p.Wait()

	log.Info().Msg("init kline success")
}

func (s *Stockx) getKline() []model.Kline {
	var data []model.Kline
	db.KlineDB.Collection(util.CodeToInt(s.Id)).Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"code": s.Id}).
		Sort(bson.M{"time": 1}).
		Do()).All(&data)
	return data
}
