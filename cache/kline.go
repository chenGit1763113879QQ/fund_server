package cache

import (
	"fund/model"
)

type KlineCache struct {
	Code   string
	Kline  []*model.Kline
	Pkline *model.PreKline
}

var Kline []*KlineCache

func New(keys []string) {
	Kline = make([]*KlineCache, len(keys))
	for i, k := range keys {
		Kline[i] = &KlineCache{Code: k}
	}
}

func LoadPKline(key string) *model.PreKline {
	for _, k := range Kline {
		if key == k.Code {
			return k.Pkline
		}
	}
	return nil
}

// Store kline and pkline
func Store(key string, value []*model.Kline) {
	// pkline
	pk := &model.PreKline{
		Time:  make([]int64, len(value)),
		Close: make([]float64, len(value)),
	}

	for i, v := range value {
		pk.Time[i] = v.Time
		pk.Close[i] = v.Close
	}

	for _, k := range Kline {
		if key == k.Code {
			k.Kline = value
			k.Pkline = pk
			return
		}
	}
}

func RangeKline(f func(k string, v []*model.Kline)) {
	for _, k := range Kline {
		f(k.Code, k.Kline)
	}
}

func RangePKline(f func(k string, v *model.PreKline)) {
	for _, k := range Kline {
		f(k.Code, k.Pkline)
	}
}

func Len() int {
	return len(Kline)
}
