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
	a := s.data[key]
	s.mu.RUnlock()
	return a
}

func (s *KlineMap) Store(key string, value []model.Kline) {
	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()
}

func (s *KlineMap) Range(f func(k string, v []model.Kline)) {
	s.mu.RLock()
	for k, v := range s.data {
		f(k, v)
	}
	s.mu.RUnlock()
}
