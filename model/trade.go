package model

import (
	"fund/util"
)

type Trade struct {
	Code    string
	Arg     float64
	ArgName string `bson:"arg_name"`

	Logs    []Tick
	ticks   []Tick
	Profits []Profit
}

type Tick struct {
	Price float64
	Time  int64
	Type  bool
}

type Profit struct {
	PctChg    float64 `bson:"pct_chg"`
	StartTime int64   `bson:"start_time"`
	EndTime   int64   `bson:"end_time"`
	Duration  int64   `bson:"duration"`
}

func NewTrade(code string, arg float64, argName string) *Trade {
	return &Trade{
		Code:    code,
		Arg:     arg,
		ArgName: argName,
		Logs:    make([]Tick, 0),
		ticks:   make([]Tick, 0),
	}
}

func (t *Trade) Buy(k *Kline) {
	tick := Tick{
		Price: k.Close,
		Time:  k.Time,
		Type:  true,
	}
	t.Logs = append(t.Logs, tick)
	t.ticks = append(t.ticks, tick)
}

func (t *Trade) Sell(k *Kline) {
	length := len(t.ticks)
	if length == 0 {
		return
	}

	holdsPrice := make([]float64, length)
	for i := range t.ticks {
		holdsPrice[i] = t.ticks[i].Price
	}

	avgPrice := util.Mean(holdsPrice)

	// profit
	t.Profits = append(t.Profits, Profit{
		PctChg:    (k.Close/avgPrice - 1) * 100,
		StartTime: t.ticks[0].Time,
		EndTime:   k.Time,
		Duration:  k.Time - t.ticks[0].Time,
	})

	// tick logs
	t.ticks = make([]Tick, 0)
	t.Logs = append(t.Logs, Tick{
		Price: k.Close,
		Time:  k.Time,
		Type:  true,
	})
}
