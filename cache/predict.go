package cache

import (
	"sync"
	"time"
)

type PreMap struct {
	close map[string][]float64
	time  map[string][]time.Time
	sync.RWMutex
}

var PreKlineMap = &PreMap{
	close: make(map[string][]float64),
	time:  make(map[string][]time.Time),
}

func (s *PreMap) Load(key string) ([]float64, []time.Time) {
	s.RLock()
	defer s.RUnlock()
	return s.close[key], s.time[key]
}

func (s *PreMap) Store(key string, close []float64, times []time.Time) {
	s.Lock()
	defer s.Unlock()
	s.close[key] = close
	s.time[key] = times
}

func (s *PreMap) Range(f func(k string, close []float64, times []time.Time)) {
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.close {
		t := s.time[k]
		f(k, v, t)
	}
}

func (s *PreMap) Clear() {
	KlineMap.Lock()
	defer KlineMap.Unlock()
	for k := range s.close {
		delete(s.close, k)
	}
	for k := range s.time {
		delete(s.time, k)
	}
}

func (s *PreMap) Len() int {
	KlineMap.RLock()
	defer KlineMap.RUnlock()
	return len(s.close)
}
