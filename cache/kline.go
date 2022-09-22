package cache

import (
	"fund/model"
	"sync"
)

type Map struct {
	data map[string][]*model.Kline
	sync.RWMutex
}

var KlineMap = &Map{data: make(map[string][]*model.Kline)}

func (s *Map) Load(key string) []*model.Kline {
	s.RLock()
	defer s.RUnlock()
	return s.data[key]
}

func (s *Map) Store(key string, value []*model.Kline) {
	s.Lock()
	defer s.Unlock()
	s.data[key] = value
}

func (s *Map) Range(f func(k string, v []*model.Kline)) {
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.data {
		f(k, v)
	}
}

func (s *Map) Clear() {
	KlineMap.Lock()
	defer KlineMap.Unlock()
	for k := range s.data {
		delete(s.data, k)
	}
}

func (s *Map) Len() int {
	KlineMap.RLock()
	defer KlineMap.RUnlock()
	return len(s.data)
}
