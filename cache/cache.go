package cache

import (
	"fund/model"
	"sync"

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
	if key == "" {
		return false
	}
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
		if v.MarketType == "CN" && v.Type == "stock" {
			f(k, v)
		}
	}
}
