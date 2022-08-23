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
	mu   sync.RWMutex
}

func (s *StockMap) Exist(key string) bool {
	if key == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[key]
	return ok
}

func (s *StockMap) Load(key string) model.Stock {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[key]
}

func (s *StockMap) Loads(keys []string) []model.Stock {
	res := make([]model.Stock, len(keys))
	s.mu.RLock()
	for i := range keys {
		res[i] = s.data[keys[i]]
	}
	s.mu.RUnlock()
	return res
}

func (s *StockMap) Store(key string, value model.Stock) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *StockMap) Range(f func(k string, v model.Stock)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.data {
		f(k, v)
	}
}
