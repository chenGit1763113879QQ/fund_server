package cache

import (
	"fund/model"

	"cloud.google.com/go/civil"
)

type KlineCache struct {
	keys   []string
	kline  [][]*model.Kline
	pkline []*model.PreKline
}

var Kline *KlineCache

func New(size int) *KlineCache {
	return &KlineCache{
		keys:   make([]string, size),
		kline:  make([][]*model.Kline, size),
		pkline: make([]*model.PreKline, size),
	}
}

func (s *KlineCache) LoadPKline(key string) *model.PreKline {
	for i, k := range s.keys {
		if key == k {
			return s.pkline[i]
		}
	}
	return nil
}

// Store kline and pkline
func (s *KlineCache) Store(key string, value []*model.Kline) {
	// pkline
	pkline := &model.PreKline{
		Time:  make([]civil.Date, len(value)),
		Open:  make([]float64, len(value)),
		Close: make([]float64, len(value)),
	}
	for i, v := range value {
		pkline.Time[i] = v.Time
		pkline.Open[i] = v.Open
		pkline.Close[i] = v.Close
	}

	for i, k := range s.keys {
		if key == k {
			s.kline[i] = value
			s.pkline[i] = pkline
			return
		}
	}
}

func (s *KlineCache) RangeKline(f func(k string, v []*model.Kline)) {
	for i, k := range s.keys {
		f(k, s.kline[i])
	}
}

func (s *KlineCache) RangePKline(f func(k string, v *model.PreKline)) {
	for i, k := range s.keys {
		f(k, s.pkline[i])
	}
}

func (s *KlineCache) Len() int {
	return len(s.keys)
}
