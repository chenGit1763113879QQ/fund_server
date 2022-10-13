package model

import (
	"fmt"
	"fund/util"
	"time"

	"github.com/mozillazg/go-pinyin"
	"go.mongodb.org/mongo-driver/bson"
)

var PinyinArg = pinyin.NewArgs()

func init() {
	PinyinArg.Fallback = func(r rune, a pinyin.Args) []string {
		return []string{string(r)}
	}
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

type Stock struct {
	MarketType util.Code `bson:"marketType"` // 市场
	Type       util.Code `bson:"type"`       // 类型

	Vol         int   `json:"volume"`                           // 成交量
	Followers   int   `json:"followers"`                        // 关注数
	LimitUpDays int   `json:"limitup_days" bson:"limitup_days"` // 涨停天数
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
	Vol       int
	Followers int
	Count     int

	Id string `bson:"_id"`

	PctChg  float64 `bson:"pct_chg"`
	Pb      float64
	PeTtm   float64 `bson:"pe_ttm"`
	Tr      float64
	PctYear float64 `bson:"pct_year"`

	Amount  float64
	Mc      float64
	Fmc     float64
	MainNet float64 `bson:"main_net"`
}

func (s *Stock) CalData(m *Market) {
	if m.Freq() == 2 {
		// add pinyin
		if util.IsChinese(s.Name) {
			for _, c := range pinyin.LazyPinyin(s.Name, PinyinArg) {
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
	PctChg float64 `bson:"pct_chg" mapstructure:"percent"`
	Tr     float64 `bson:",omitempty" mapstructure:"turnoverrate"`

	Pe  float64 `bson:",omitempty"`
	Pb  float64 `bson:",omitempty"`
	Ps  float64 `bson:",omitempty"`
	Pcf float64 `bson:",omitempty"`

	Time      time.Time
	TimeStamp int64 `bson:"-"`

	Code       string
	WeightAvg  float64 `bson:"weight_avg,omitempty"`
	WinnerRate float64 `bson:"winner_rate,omitempty"`

	Open   float64
	High   float64
	Low    float64
	Close  float64
	Vol    int64 `mapstructure:"volume"`
	Amount float64

	MainNet float64 `bson:",omitempty"`

	MACD     float64 `bson:",omitempty"`
	MACD_DEA float64 `bson:",omitempty" mapstructure:"dea"`
	MACD_DIF float64 `bson:",omitempty" mapstructure:"dif"`

	BOLL_MID float64 `bson:",omitempty" mapstructure:"ma20"`
	BOLL_UP  float64 `bson:",omitempty" mapstructure:"ub"`
	BOLL_LOW float64 `bson:",omitempty" mapstructure:"lb"`

	Mc      float64 `bson:",omitempty" mapstructure:"market_capital"`
	Balance float64 `bson:",omitempty"`

	HoldVolCN   int64   `bson:"hold_vol_cn,omitempty" mapstructure:"hold_volume_cn"`
	HoldRatioCN float64 `bson:"hold_ratio_cn,omitempty" mapstructure:"hold_ratio_cn"`
	NetVolCN    int64   `bson:"net_vol_cn,omitempty" mapstructure:"net_volume_cn"`

	HoldVolHK   int64   `bson:"hold_vol_hk,omitempty" mapstructure:"hold_volume_hk"`
	HoldRatioHK float64 `bson:"hold_ratio_hk,omitempty" mapstructure:"hold_ratio_hk"`
	NetVolHK    int64   `bson:"net_vol_hk,omitempty" mapstructure:"net_volume_hk"`
}

type MinuteKline struct {
	Code      string
	TradeDate time.Time `bson:"trade_date"`
	Time      []int64   `bson:"time" mapstructure:"tick_at"`
	Open      []float64 `mapstructure:"open_px"`
	Close     []float64 `mapstructure:"close_px`
	Avg       []float64 `mapstructure:"avg_px`
}

type PredictRes struct {
	Period    int
	Limit     int
	Std       float64
	SrcCode   string    `bson:"src_code"`
	MatchCode string    `bson:"match_code"`
	StartDate time.Time `bson:"start_date"`
	PreDirect bool      `bson:"pre_direct"`  // 预测方向 true为涨
	PrePctChg float64   `bson:"pre_pct_chg"` // 预测涨幅
}

type PredictArr []*PredictRes

func (x PredictArr) Len() int {
	return len(x)
}
func (x PredictArr) Less(i, j int) bool {
	return x[i].Std < x[j].Std
}
func (x PredictArr) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}
