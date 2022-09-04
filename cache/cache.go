package cache

import (
	"context"
	"fmt"
	"fund/model"
	"fund/util"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

var (
	Stock = &StockMap{data: make(map[string]model.Stock)}

	Numbers    sync.Map
	MainFlow   any
	NorthMoney any
	MarketHot  []bson.M
)

type StockMap struct {
	data map[string]model.Stock
	sync.RWMutex
}

func (s *StockMap) Exist(key string) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.data[key]
	return ok
}

func (s *StockMap) Load(key string) model.Stock {
	s.RLock()
	defer s.RUnlock()
	return s.data[key]
}

func (s *StockMap) Loads(keys []string) []model.Stock {
	res := make([]model.Stock, len(keys))
	s.RLock()
	for i := range keys {
		res[i] = s.data[keys[i]]
	}
	s.RUnlock()
	return res
}

func (s *StockMap) Store(key string, value model.Stock) {
	s.Lock()
	defer s.Unlock()
	s.data[key] = value
}

func (s *StockMap) Stores(key []string, value []model.Stock) {
	if len(key) != len(value) {
		fmt.Println("not equal length between keys and values")
	}
	s.Lock()
	defer s.Unlock()
	for i := range key {
		if value[i].Price > 0 {
			s.data[key[i]] = value[i]
		}
	}
}

func (s *StockMap) Range(f func(k string, v model.Stock)) {
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.data {
		f(k, v)
	}
}

func (s *StockMap) RangeForCNStock(f func(k string, v model.Stock)) {
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.data {
		if v.MarketType == util.MARKET_CN && v.Type == util.TYPE_STOCK {
			f(k, v)
		}
	}
}

func (s *StockMap) Watch(ctx context.Context, onWatch func(any), keys []string) {
	values := s.Loads(keys)
	for {
		select {
		case <-ctx.Done():
			return

		default:
			newValues := s.Loads(keys)
			for i := range values {
				if newValues[i] != values[i] {
					onWatch(newValues[i])
					values[i] = newValues[i]
				}
			}
		}
		time.Sleep(time.Millisecond / 10)
	}
}
