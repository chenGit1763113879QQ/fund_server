package model

import (
	"fund/util"

	"github.com/mozillazg/go-pinyin"
)

var PinyinArg = pinyin.NewArgs()

type Index struct {
	Market struct {
		Region     string `json:"region"`
		Status     string
		StatusName string `json:"status"`
		TimeZone   string `json:"time_zone"`
	} `json:"market"`

	Stock Stock `json:"quote"`
}

type Basic struct {
	Id         string    `bson:"_id"`                   // 代码
	Pinyin     string    `bson:"pinyin,omitempty"`      // 拼音
	LazyPinyin string    `bson:"lazy_pinyin,omitempty"` // 简单拼音
	MarketType util.Code `bson:"marketType,omitempty"`  // 市场
	Type       util.Code `bson:"type,omitempty"`        // 类型
}

func (s *Basic) AddPinYin(name string) {
	if util.IsChinese(name) {
		s.Pinyin = ""
		for _, c := range pinyin.LazyPinyin(name, PinyinArg) {
			s.Pinyin += c
			s.LazyPinyin += string(c[0])
		}
	}
}

type Stock struct {
	Basic `bson:",inline"`

	Vol         int   `json:"volume"`                                     // 成交量
	Followers   int   `json:"followers"`                                  // 关注数
	LimitUpDays int   `json:"limitup_days" bson:"limitup_days,omitempty"` // 涨停天数
	Time        int64 `json:"time" bson:"time,omitempty"`

	Symbol string `json:"symbol"`
	Name   string `json:"name"`

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
	Basic `bson:",inline"`

	Vol       int64
	Followers int64
	Count     int64

	Symbol string `json:"encode"`
	Name   string `json:"name" bson:",omitempty"`

	PctChg  float64 `bson:"pct_chg"`
	Pb      float64
	PeTtm   float64 `bson:"pe_ttm"`
	Tr      float64
	PctYear float64 `bson:"pct_year"`

	Amount  float64
	Mc      float64
	Fmc     float64
	MainNet float64  `bson:"main_net"`
	Members []string `bson:"members,omitempty"`
}

func (s *Stock) CalData(m *Market) {
	if m.Freq() == 2 {
		s.AddPinYin(s.Name)
	}

	s.Id = util.ParseCode(s.Symbol)
	s.MarketType = m.Market
	s.Type = m.Type

	if s.Vol > 0 {
		s.Avg = s.Amount / float64(s.Vol)
	}
}

type Kline struct {
	Time int64 `mapstructure:"timestamp"`
	Vol  int64 `mapstructure:"volume"`

	HoldVolCN int64 `bson:"hold_vol_cn,omitempty" mapstructure:"hold_volume_cn"`
	NetVolCN  int64 `bson:"net_vol_cn,omitempty" mapstructure:"net_volume_cn"`
	HoldVolHK int64 `bson:"hold_vol_hk,omitempty" mapstructure:"hold_volume_hk"`
	NetVolHK  int64 `bson:"net_vol_hk,omitempty" mapstructure:"net_volume_hk"`

	PctChg float64 `bson:"pct_chg" mapstructure:"percent"`
	Tr     float64 `bson:",omitempty" mapstructure:"turnoverrate"`

	Pe  float64 `bson:",omitempty"`
	Pb  float64 `bson:",omitempty"`
	Ps  float64 `bson:",omitempty"`
	Pcf float64 `bson:",omitempty"`

	Open   float64
	High   float64
	Low    float64
	Close  float64
	Amount float64

	MainNet float64 `bson:",omitempty"`

	BOLL_MID float64 `bson:",omitempty" mapstructure:"ma20"`
	BOLL_UP  float64 `bson:",omitempty" mapstructure:"ub"`
	BOLL_LOW float64 `bson:",omitempty" mapstructure:"lb"`
	Balance  float64 `bson:",omitempty"`

	HoldRatioCN float64 `bson:"hold_ratio_cn,omitempty" mapstructure:"hold_ratio_cn"`
	HoldRatioHK float64 `bson:"hold_ratio_hk,omitempty" mapstructure:"hold_ratio_hk"`

	Code string

	// 过去30日信息
	Stat KlineStat
}

type KlineStat struct {
	PctChg    uint64 `bson:"pct_chg"`     // pct_chg > 0 ? 1 : 0
	MainNet   uint64 `bson:"main_net"`    // main_net > 0 ? 1 : 0
	HKHoldNet uint64 `bson:"hk_hold_net"` // hk_hold_net > 0 ? 1 : 0
}
