package model

import (
	"fund/util"
	"time"
)

type Market struct {
	Market util.Code `json:"-"`
	Type   util.Code `json:"-"`
	count  uint8

	Status bool
	StatusName string
	TradeTime  time.Time
}

// get current trade_date string. exp: 2022/09/22
func (m *Market) ParseTradeDate() string {
	return m.TradeTime.Format("2006/01/02")
}

func (m *Market) Incr() {
	m.count++
	if m.count > 100 {
		m.count = m.count % 100
	}
}

func (m *Market) Freq() int {
	if m.count%100 == 0 {
		return 2
	}
	if m.count%10 == 0 {
		return 1
	}
	return 0
}

func (m *Market) ReSet() {
	m.count = 0
}

func (m *Market) FreqIsZero() bool {
	return m.count == 0
}
