package model

import (
	"fmt"
	"fund/util"
	"time"

	"github.com/mozillazg/go-pinyin"
	"go.mongodb.org/mongo-driver/bson"
)

var pinyinArg = pinyin.NewArgs()

type Market struct {
	Market uint8
	Type   uint8
	count  uint8

	Status bool

	StrMarket string
	StrType   string

	StatusName string // choice: [闭市, 盘前交易, 集合竞价，交易中，休市]
	TradeTime  time.Time
}

type Index struct {
	Market struct {
		Region     string `json:"region"`
		Status     string
		StatusName string `json:"status"`
		TimeZone   string `json:"time_zone"`
	} `json:"market"`

	Stock Stock `json:"quote"`
}

func init() {
	pinyinArg.Fallback = func(r rune, a pinyin.Args) []string {
		return []string{string(r)}
	}
}

func (m *Market) ReSet() {
	m.count = 0
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

func (m *Market) FreqIsZero() bool {
	return m.count == 0
}

type Stock struct {
	MarketType uint8 `bson:"marketType"` // 市场
	Type       uint8 `bson:"type"`       // 类型

	Vol         uint  `json:"volume"`                                     // 成交量
	Followers   int   `json:"followers"`                                  // 关注数
	LimitUpDays int   `json:"limitup_days" bson:"limitup_days,omitempty"` // 涨停天数
	Time        int64 `json:"time" bson:"time,omitempty"`

	Id     string `bson:"_id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`

	Pinyin     string `bson:"pinyin,omitempty"`      // 拼音
	LazyPinyin string `bson:"lazy_pinyin,omitempty"` // 简单拼音

	PctChg float64 `json:"percent" bson:"pct_chg"` // 涨跌幅
	Amp    float64 `json:"amplitude"`              // 振幅
	Tr     float64 `json:"turnover_rate"`          // 换手率
	Vr     float64 `json:"volume_ratio"`           // 量比

	Pct5m    float64 `json:"percent5m" bson:"pct_5m"`              // 5分钟涨幅
	PctYear  float64 `json:"current_year_percent" bson:"pct_year"` // 今年涨幅
	TotalPct float64 `json:"total_percent" bson:"total_pct"`       // 至今涨幅

	Pb    float64 `json:"pb"`                   // 市净率
	PeTtm float64 `json:"pe_ttm" bson:"pe_ttm"` // 市盈ttm
	Ps    float64 `json:"ps"`                   // 市销率
	Eps   float64 `json:"eps"`                  // 每股收益
	Roe   float64 `json:"roe_ttm"`              // ROE
	Dv    float64 `json:"dividend_yield"`       // 股息率
	Pcf   float64 `json:"pcf"`                  // ?

	IncomeYoy    float64 `json:"income_cagr"`     // 营收同比
	NetProfitYoy float64 `json:"net_profit_cagr"` // 净利润同比

	Price  float64 `json:"current"` // 价格
	Amount float64 `json:"amount"`  // 成交额
	Avg    float64 // 均价

	Mc         float64 `json:"market_capital"`                  // 总市值
	Fmc        float64 `json:"float_market_capital"`            // 流通市值
	TotalShare float64 `json:"total_shares" bson:"total_share"` // 总股本
	FloatShare float64 `json:"float_shares" bson:"float_share"` // 流通股本

	MainNet  float64 `json:"main_net_inflows" bson:"main_net"`  // 主力净流入
	NorthNet float64 `json:"north_net_inflow" bson:"north_net"` // 北向资金净流入
}

type Industry struct {
	MarketType uint8 `bson:"marketType,omitempty"`
	Type       uint8 `bson:"type,omitempty"`
	Vol        uint  `bson:"vol"`

	Id         string `bson:"_id"`
	Name       string `bson:"name"`
	Pinyin     string `bson:"pinyin,omitempty"`
	LazyPinyin string `bson:"lazy_pinyin,omitempty"`

	PctChg  float64 `bson:"pct_chg"`
	Pb      float64 `bson:"pb"`
	PeTtm   float64 `bson:"pe_ttm"`
	Tr      float64 `bson:"tr"`
	PctYear float64 `bson:"pct_year"`

	Amount  float64 `bson:"amount"`
	Mc      float64 `bson:"mc"`
	Fmc     float64 `bson:"fmc"`
	MainNet float64 `bson:"main_net"`

	ConnList  []Stk `bson:"c,omitempty"`
	PctLeader Stk   `bson:"pct_leader"`
}

// 成分股
type Stk struct {
	Code   string  `bson:"code"`
	Name   string  `bson:"name"`
	PctChg float64 `bson:"pct_chg"`
}

func (s *Stock) CalData(m *Market) {
	if m.Freq() == 2 {
		// add pinyin
		if util.IsChinese(s.Name) {
			for _, c := range pinyin.LazyPinyin(s.Name, pinyinArg) {
				s.Pinyin += c
				s.LazyPinyin += string(c[0])
			}
		}
	}

	s.Id = s.Symbol
	s.MarketType = m.Market
	s.Type = m.Type

	// format code
	switch m.Market {
	case util.MARKET_CN:
		s.Id = fmt.Sprintf("%s.%s", s.Id[2:], s.Id[0:2])

	case util.MARKET_HK:
		if s.Id[0:2] == "HK" {
			s.Id = s.Id[2:]
		}
		s.Id += ".HK"

	case util.MARKET_US:
		if s.Id[0] == '.' {
			s.Id = s.Id[1:]
		}
		s.Id += ".US"
	}

	if s.Vol > 0 {
		s.Avg = s.Amount / float64(s.Vol)
	}
}

type Groups struct {
	Groups []Group  `json:"groups"`
	Stocks []bson.M `json:"stocks"`
}

type Group struct {
	IsActive bool `json:"isActive"`
	IsSys    bool `json:"isSys"`

	Name   string `json:"name" binding:"max=6"`
	Market string `json:"market,omitempty" bson:",omitempty"`

	Parent []string `json:"parent,omitempty" bson:",omitempty" binding:"max=3,unique"`
	List   []string `json:"list"`

	MarketRange []Range `json:"market_range" bson:",omitempty"`
	FinaRange   []Range `json:"fina_range" bson:",omitempty"`
}

type Range struct {
	Name  string  `json:"name"`
	Left  float64 `json:"left"`
	Right float64 `json:"right"`
	Unit  float64 `json:"unit"`
}

type Kline struct {
	PctChg float64 `bson:"pct_chg" csv:"percent"`
	Tr     float64 `bson:",omitempty" csv:"turnoverrate"`

	Pe float64 `bson:",omitempty" csv:"pe"`
	Pb float64 `bson:",omitempty" csv:"pb"`

	Time      time.Time `bson:"time" csv:"-"`
	TimeStamp int64     `bson:"-" csv:"timestamp"`
	Code      string    `bson:"code"`

	Open   float64 `csv:"open"`
	High   float64 `csv:"high"`
	Low    float64 `csv:"low"`
	Close  float64 `csv:"close"`
	Vol    int64   `csv:"volume"`
	Amount float64 `csv:"amount"`

	MainNet float64 `bson:",omitempty"`
	Net     float64 `bson:",omitempty"`

	Dv float64 `bson:"dv_ttm,omitempty"`

	KDJ_K float64 `bson:"kdj_k" csv:"kdjk"`
	KDJ_D float64 `bson:"kdj_d" csv:"kdjd"`
	KDJ_J float64 `bson:"kdj_j" csv:"kdjj"`

	WinnerRate float64 `bson:"winner_rate"`

	RSI6  float64 `bson:"rsi_6" csv:"rsi1"`
	RSI12 float64 `bson:"rsi_12" csv:"rsi2"`
	RSI24 float64 `bson:"rsi_24" csv:"rsi3"`

	MACD     float64 `bson:"macd" csv:"macd"`
	MACD_DEA float64 `bson:"macd_dea" csv:"dea"`
	MACD_DIF float64 `bson:"macd_dif" csv:"dif"`

	BOLL_MID float64 `bson:"boll_mid" csv:"ma20"`
	BOLL_UP  float64 `bson:"boll_upper" csv:"ub"`
	BOLL_LOW float64 `bson:"boll_lower" csv:"lb"`

	CCI     float64 `csv:"cci"`
	Balance float64 `bson:",omitempty" csv:"balance"`

	HoldVolCN   int64   `bson:"hold_vol_cn,omitempty" csv:"hold_volume_cn"`
	HoldRatioCN float64 `bson:"hold_ratio_cn,omitempty" csv:"hold_ratio_cn"`
	NetVolCN    int64   `bson:"net_vol_cn,omitempty" csv:"net_volume_cn"`

	HoldVolHK   int64   `bson:"hold_vol_hk,omitempty" csv:"hold_volume_hk"`
	HoldRatioHK float64 `bson:"hold_ratio_hk,omitempty" csv:"hold_ratio_hk"`
	NetVolHK    int64   `bson:"net_vol_hk,omitempty" csv:"net_volume_hk"`
}
