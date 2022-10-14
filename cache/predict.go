package cache

import (
	"sync"

	"cloud.google.com/go/civil"
)

type PreData struct {
	Time  []civil.Date
	Close []float64
	Open  []float64
}

func (s *PreData) Len() int {
	return len(s.Close)
}

type PreMap struct {
	keys []string
	data []PreData
	sync.Mutex
}

var PreKlineMap = &PreMap{
	keys: make([]string, 0),
	data: make([]PreData, 0),
}

func (s *PreMap) Load(key string) PreData {
	for i := range s.keys {
		if s.keys[i] == key {
			return s.data[i]
		}
	}
	return PreData{}
}

func (s *PreMap) Store(key string, value PreData) {
	s.Lock()
	defer s.Unlock()
	// find
	for i := range s.keys {
		if s.keys[i] == key {
			s.data[i] = value
			return
		}
	}
	// unfind
	s.keys = append(s.keys, key)
	s.data = append(s.data, value)
}

func (s *PreMap) Range(f func(k string, value PreData)) {
	for i := range s.keys {
		f(s.keys[i], s.data[i])
	}
}

func (s *PreMap) Clear() {
	s.Lock()
	defer s.Unlock()
	s.keys = make([]string, 0)
	s.data = make([]PreData, 0)
}

func (s *PreMap) Len() int {
	return len(s.data)
}
