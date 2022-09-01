package model

import (
	"fund/util"
	"time"
)

type Trade struct {
	Title string

	Arg     float64
	ArgName string

	head   *Tick
	Profit Profit
}

type Tick struct {
	Price float64
	Time  time.Time
	Next  *Tick
}

type Profit struct {
	PctChg    float64       `bson:"pct_chg"`
	StartTime time.Time     `bson:"start_time"`
	EndTime   time.Time     `bson:"end_time"`
	Duration  time.Duration `bson:"duration"`
}

func NewTrade(title string, arg float64) *Trade {
	return &Trade{Title: title, Arg: arg}
}

func (t *Trade) Buy(k Kline) {
	p := t.head
	for p != nil {
		p = p.Next
	}
	p = &Tick{
		Price: k.Close,
		Time:  k.Time,
	}
}

func (t *Trade) Sell(k Kline) {
	if t.head == nil {
		return
	}
	holdsPrice := make([]float64, 0)
	holdsTime := make([]int64, 0)

	p := t.head
	for {
		holdsPrice = append(holdsPrice, t.head.Price)
		holdsTime = append(holdsTime, t.head.Time.Unix())
		if p.Next == nil {
			break
		}
		p = p.Next
	}

	avgPrice := util.Mean(holdsPrice)
	avgTime := util.Mean(holdsTime)
	pctChg := k.Close / avgPrice

	t.Profit = Profit{
		PctChg:    pctChg,
		StartTime: t.head.Time,
		EndTime:   p.Time,
		Duration:  time.Duration(avgTime),
	}
}
