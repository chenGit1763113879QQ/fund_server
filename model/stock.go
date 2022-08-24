package model

import (
	"fund/util"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mozillazg/go-pinyin"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	Params = []KlineParams{
		{"day", "2006-01-02", 101},
		{"min", "2006-01-02 15:04", 5},
	}
	pinyinArg = pinyin.NewArgs()
)

type Market struct {
	Status     bool
	count      uint8
	Size       uint
	Name       string
	StatusName string // choice: [闭市, 盘前交易, 集合竞价，交易中，休市]
	Type       string
	Fs         string
	TradeTime  time.Time
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
	if m.count > 200 {
		m.count = m.count % 200
	}
}

func (m *Market) Freq() int {
	if m.count%200 == 0 {
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

// freq: 0高频 1中频 2低频 (默认0)
type Stock struct {
	Id         string `json:"f12" bson:"_id"`                     // 代码
	Cid        string `bson:"cid,omitempty"`                      // cid
	Name       string `json:"f14" bson:"name,omitempty" freq:"2"` // 名称
	MarketType string `bson:"marketType,omitempty"`               // 市场
	Type       string `bson:"type,omitempty"`                     // 类型

	Pinyin     string `bson:"pinyin,omitempty"`      // 拼音
	LazyPinyin string `bson:"lazy_pinyin,omitempty"` // 简单拼音

	CidOld int `json:"f13" bson:"-" freq:"2"`   // cid
	Vol    int `json:"f5" bson:"vol,omitempty"` // 成交量
	Buy    int `json:"f34" bson:"-"`            // 买盘
	Sell   int `json:"f35" bson:"-"`            // 卖盘

	Price  float64 `json:"f2" bson:"price,omitempty"`           // 价格
	High   float64 `json:"f15" bson:"high,omitempty" freq:"1"`  // 最高
	Low    float64 `json:"f16" bson:"low,omitempty" freq:"1"`   // 最低
	Open   float64 `json:"f17" bson:"open,omitempty" freq:"1"`  // 开盘
	Close  float64 `json:"f18" bson:"close,omitempty" freq:"1"` // 收盘
	PctChg float64 `json:"f3" bson:"pct_chg"`                   // 涨跌幅
	Amount float64 `json:"f6" bson:"amount,omitempty"`          // 成交额
	Avg    float64 `bson:"avg,omitempty"`                       // 均价

	Mc         float64 `json:"f20" bson:"mc,omitempty" freq:"1"`          // 总市值
	Fmc        float64 `json:"f21" bson:"fmc,omitempty" freq:"1"`         // 流通市值
	TotalShare float64 `json:"f38" bson:"total_share,omitempty" freq:"2"` // 总股本
	FloatShare float64 `json:"f39" bson:"float_share,omitempty" freq:"2"` // 流通股本

	Pb       float64 `json:"f23" bson:"pb,omitempty" freq:"1"`      // 市净率
	PeTtm    float64 `json:"f115" bson:"pe_ttm,omitempty" freq:"1"` // 市盈ttm
	Amp      float64 `json:"f7" bson:"amp,omitempty" freq:"1"`      // 振幅
	PctRate  float64 `json:"f22" bson:"pct_rate"`                   // 涨速
	Tr       float64 `json:"f8" bson:"tr,omitempty" freq:"1"`       // 换手率
	Wb       float64 `json:"f33" bson:"wb"`                         // 委比
	Vr       float64 `json:"f10" bson:"vr,omitempty" freq:"1"`      // 量比
	Pct5min  float64 `json:"f11" bson:"pct_5min"`                   // 5分钟涨幅
	Pct60day float64 `json:"f24" bson:"pct_60day"`                  // 60日涨幅
	PctYear  float64 `json:"f25" bson:"pct_year"`                   // 今年涨幅

	Net       float64 `bson:"net,omitempty"` // 净流入
	MainHuge  float64 `json:"f66" bson:"main_huge,omitempty"`
	MainBig   float64 `json:"f72" bson:"main_big,omitempty"`
	MainMid   float64 `json:"f78" bson:"main_mid,omitempty"`
	MainSmall float64 `json:"f84" bson:"main_small,omitempty"`
	MainNet   float64 `bson:"main_net,omitempty"`                            // 主力净流入
	Day3Net   float64 `json:"f267" bson:"3day_main_net,omitempty" freq:"1"`  // 3日净流入
	Day5Net   float64 `json:"f164" bson:"5day_main_net,omitempty" freq:"1"`  // 5日净流入
	Day10Net  float64 `json:"f174" bson:"10day_main_net,omitempty" freq:"1"` // 10日净流入
}

type Industry struct {
	Id         string `bson:"_id"`
	Name       string `bson:"name,omitempty"`
	MarketType string `bson:"marketType,omitempty"`
	Type       string `bson:"type,omitempty"`

	Pinyin     string `bson:"pinyin,omitempty"`
	LazyPinyin string `bson:"lazy_pinyin,omitempty"`

	Vol int `bson:"vol,omitempty"`

	Price     float64 `bson:"price,omitempty"`
	High      float64 `bson:"high,omitempty"`
	Low       float64 `bson:"low,omitempty"`
	Open      float64 `bson:"open,omitempty"`
	Close     float64 `bson:"close,omitempty"`
	PctChg    float64 `bson:"pct_chg"`
	Amount    float64 `bson:"amount,omitempty"`
	Pb        float64 `bson:"pb,omitempty"`
	PeTtm     float64 `bson:"pe_ttm,omitempty"`
	Tr        float64 `bson:"tr,omitempty"`
	Wb        float64 `bson:"wb"`
	Mc        float64 `bson:"mc,omitempty"`
	Fmc       float64 `bson:"fmc,omitempty"`
	Net       float64 `bson:"net"`
	MainHuge  float64 `bson:"main_huge"`
	MainBig   float64 `bson:"main_big"`
	MainMid   float64 `bson:"main_mid"`
	MainSmall float64 `bson:"main_small"`
	MainNet   float64 `bson:"main_net"`
	PctYear   float64 `bson:"pct_year"`

	ConnList      []Stk `bson:"c,omitempty"`
	PctLeader     Stk   `bson:"pct_leader"`
	MainNetLeader Stk   `bson:"main_net_leader"`
}

// 成分股
type Stk struct {
	Name    string  `bson:"name"`
	PctChg  float64 `bson:"pct_chg"`
	MainNet float64 `bson:"main_net"`
}

type AHCompare struct {
	CNCode string `json:"symbol_cn" bson:"cn_code"`
	CNName string `json:"name_cn" bson:"cn_name"`
	HKCode string `json:"symbol_hk" bson:"hk_code"`
	HKName string `json:"name_hk" bson:"hk_name"`

	CNPrice  float64 `json:"current_cn" bson:"cn_price"`
	CNPctChg float64 `json:"percent_cn" bson:"cn_pct_chg"`
	HKPrice  float64 `json:"current_hk" bson:"hk_price"`
	HKPctChg float64 `json:"percent_hk" bson:"hk_pct_chg"`

	Premium float64 `json:"premium"`
}

func (a *AHCompare) ParseId() {
	a.CNCode = a.CNCode[2:] + "." + a.CNCode[0:2]
	a.HKCode += ".HK"
}

func (s *Stock) CalData(m *Market) {
	if m.Freq() == 2 {
		s.Cid = strconv.Itoa(s.CidOld) + "." + s.Id
		if m.Name == "CN" {
			s.Name = strings.Replace(s.Name, " ", "", -1)
		}
		s.MarketType = m.Name
		s.Type = m.Type

		if util.IsChinese(s.Name) {
			for _, c := range pinyin.LazyPinyin(s.Name, pinyinArg) {
				s.Pinyin += c
				s.LazyPinyin += string(c[0])
			}
		}
	}
	s.formatId(m)

	if s.Vol > 0 {
		s.Avg = s.Amount / float64(s.Vol)
		s.Net = s.Avg * float64(s.Buy-s.Sell)
		if m.Name == "CN" {
			s.Avg /= 100
		}
	}
	s.MainNet = s.MainHuge + s.MainBig
}

// get fields
func (s *Stock) GetJsonFields(freq int) []string {
	arr := make([]string, 0)
	types := reflect.TypeOf(s).Elem()

	for i := 0; i < types.NumField(); i++ {
		tag := types.Field(i).Tag

		if tag.Get("json") != "" {
			if strconv.Itoa(freq) >= tag.Get("freq") || tag.Get("freq") == "" {
				arr = append(arr, tag.Get("json"))
			}
		}
	}
	return arr
}

// format code
func (s *Stock) formatId(m *Market) {
	switch m.Name {
	case "CN":
		switch m.Type {
		case "stock":
			switch s.Id[0] {
			case '6':
				s.Id += ".SH"
			case '0', '3':
				s.Id += ".SZ"
			case '8', '4':
				s.Id += ".BJ"
			}
		case "index":
			switch s.Id[0] {
			case '0':
				s.Id += ".SH"
			case '3':
				s.Id += ".SZ"
			}
		case "fund":
			switch s.Id[0] {
			case '5':
				s.Id += ".SH"
			case '1':
				s.Id += ".SZ"
			}

		default:
			// drop
			s.Price = 0
		}
	default:
		s.Id += "." + m.Name
	}
}

type CListOpt struct {
	Parent     string   `form:"parent"`
	MarketType string   `form:"marketType" binding:"oneof=CN HK US,omitempty"`
	Sort       string   `form:"sort"`
	Chart      string   `form:"chart"`
	Page       int64    `form:"page" binding:"min=1,omitempty"`
	List       []string `form:"list" json:"list" bson:"list" binding:"unique,omitempty"`
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

type KlineOpt struct {
	Code      string `form:"code"`
	Period    string `form:"period"`
	StartDate string `form:"start_date"`
	Select    string `form:"select"`
	Head      int    `form:"head"`
	Tail      int    `form:"tail"`
}

type KlineParams struct {
	Period string
	Format string
	Params int
}

type Kline struct {
	Time time.Time `bson:"time"`

	Close float64 `bson:"close_qfq"`

	PctChg float64 `bson:"pct_chg"`
	Amount float64
	Tr     float64

	MainNet float64
	Net     float64

	Pe float64 `bson:"pe_ttm"`
	Pb float64
	Dv float64 `bson:"dv_ttm"`

	//KDJ_K float64 `bson:"kdj_k"`
	//KDJ_D float64 `bson:"kdj_d"`
	KDJ_J float64 `bson:"kdj_j"`

	WinnerRate float64 `bson:"winner_rate"`

	//RSI6  float64 `bson:"rsi_6"`
	//RSI12 float64 `bson:"rsi_12"`
	//RSI24 float64 `bson:"rsi_24"`

	//MACD float64 `bson:"macd"`
	//MACD_DEA float64 `bson:"macd_dea"`
	//MACD_DIF float64 `bson:"macd_dif"`

	//BOLL_MID float64 `bson:"boll_mid"`
	//BOLL_UP float64 `bson:"boll_upper"`
	//BOLL_LOW float64 `bson:"boll_lower"`

	CCI float64
}
