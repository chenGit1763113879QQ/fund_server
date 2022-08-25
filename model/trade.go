package model

import (
	"fmt"
	"time"
)

// 交易模块
type Trade struct {
	title string

	holds []*Hold
	ticks []*Tick
}

// 持仓
type Hold struct {
	Time  time.Time
	Price float64
	Vol   float64
}

// 交易记录
type Tick struct {
	Id        string
	BeginTime time.Time
	EndTime   time.Time
	Profit    float64
}

func NewTrade(title string) *Trade {
	return &Trade{
		title: title,
		holds: make([]*Hold, 0),
		ticks: make([]*Tick, 0),
	}
}

// 初始化
func (s *Trade) Init() {
	s.holds = make([]*Hold, 0)
}

// 买入
func (s *Trade) Buy(k Kline) {
	if len(s.holds) > 3 {
		return
	}
	s.holds = append(s.holds, &Hold{
		Time: k.Time, Price: k.Close, Vol: 1,
	})
}

// 卖出
func (s *Trade) Sell(k Kline, id string) {
	var vol, amount float64

	for _, r := range s.holds {
		vol += r.Vol
		amount += r.Vol * r.Price
	}

	if amount == 0 {
		return
	}

	s.ticks = append(s.ticks, &Tick{
		Id:        id,
		BeginTime: s.holds[0].Time,
		EndTime:   k.Time,
		// 收益
		Profit: (k.Close*vol/amount - 1) * 100,
	})
	
	s.Init()
}

// 统计信息
func (s *Trade) RecordsInfo() {
	var profit, dur float64

	length := float64(len(s.ticks))

	for _, r := range s.ticks {
		profit += r.Profit
		dur += r.EndTime.Sub(r.BeginTime).Hours() / 24
	}

	profit /= length
	dur /= length

	fmt.Printf("回测函数%s: 样本:%.0f 周期:%.0f天 日均收益率:%.3f\n", s.title, length, dur, profit/dur)
}
