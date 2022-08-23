package pro

import (
	"fund/model"
	"sync"
)

var klineMap = &KlineMap{data: make(map[string][]model.Kline)}

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
