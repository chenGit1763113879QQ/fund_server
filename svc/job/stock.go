package job

import (
	"context"
	"fmt"
	"fund/db"
	"fund/model"
	"fund/util"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

const XQHOST = "https://xueqiu.com/service/v5/stock"

var (
	ctx = context.Background()

	Markets = []*model.Market{
		{Market: util.MARKET_CN, Type: util.TYPE_STOCK, StrMarket: "CN", StrType: "sh_sz"},
		{Market: util.MARKET_HK, Type: util.TYPE_STOCK, StrMarket: "HK", StrType: "hk"},
		{Market: util.MARKET_US, Type: util.TYPE_STOCK, StrMarket: "US", StrType: "us"},
	}

	Cond = sync.NewCond(&sync.Mutex{})
)

func init() {
	getMarketStatus()
	log.Info().Msg("init market status success.")

	for _, p := range Markets {
		go getRealStock(p)
	}
	go getNews()

	util.GoJob(getMarketStatus, time.Second)
}

func GetTradeTime(code string) time.Time {
	splits := strings.Split(code, ".")
	if len(splits) == 2 {
		switch splits[1] {
		case "SH", "SZ", "BJ", "TI":
			return Markets[0].TradeTime
		case "HK":
			return Markets[1].TradeTime
		case "US":
			return Markets[2].TradeTime
		}
	}
	return time.Unix(0, 0)
}

func getRealStock(m *model.Market) {
	url := fmt.Sprintf("%s/screener/quote/list?size=5000&order_by=amount&type=%s", XQHOST, m.StrType)

	for {
		freq := m.Freq()

		if freq == 2 && m.Type == util.TYPE_STOCK {
			log.Info().Msgf("update stock[%s]", m.StrType)
		}

		body, err := util.GetAndRead(url)
		if err != nil {
			continue
		}

		var data []model.Stock
		util.UnmarshalJSON(body, &data, "data", "list")

		bulk := db.Stock.Bulk()

		for i := range data {
			data[i].CalData(m)

			if data[i].Price > 0 {
				// update db
				bulk.UpdateId(data[i].Id, bson.M{"$set": data[i]})

				// insert db
				if freq == 2 {
					db.Stock.InsertOne(ctx, data[i])
				}
			}
		}

		bulk.Run(ctx)
		updateMinute(data, m)

		if freq >= 1 {
			go getDistribution(m)
			go getIndustry(m)

			if m.Market == util.MARKET_CN {
				go getMainFlow()
				go getNorthMoney()
			}
		}
		Cond.Broadcast()
		m.Incr()

		for !m.Status {
			time.Sleep(time.Millisecond * 100)
			m.ReSet()
		}
		time.Sleep(time.Millisecond * 500)
	}
}

func updateMinute(s []model.Stock, m *model.Market) {
	tradeTime := m.TradeTime.Format("2006/01/02 15:04")
	date := strings.Split(tradeTime, " ")[0]

	newTime, _ := time.Parse("2006/01/02 15:04", tradeTime)

	if m.FreqIsZero() {
		db.MinuteDB.CreateCollection(ctx, date)
		db.MinuteDB.Collection(date).EnsureIndexes(ctx, []string{"_id.code,_id.time"}, nil)
	}

	a := time.Now()
	if a.Second() > 15 && a.Second() < 45 {
		return
	}

	bulk := db.MinuteDB.Collection(date).Bulk()
	for _, i := range s {
		if i.Price > 0 {
			bulk.UpsertId(
				bson.M{"code": i.Id, "time": newTime.Unix()},
				bson.M{"price": i.Price, "pct_chg": i.PctChg, "vol": i.Vol, "avg": i.Avg,
					"main_net": i.MainNet, "minutes": newTime.Minute()},
			)
		}
	}
	go bulk.Run(ctx)
}

func InitKlines() {
	var stocks []struct {
		Id     string `bson:"_id"`
		Symbol string
	}
	db.Stock.Find(ctx, bson.M{}).All(&stocks)

	p := util.NewPool(5)
	for _, i := range stocks {
		p.NewTask(func() {
			klines := getKline(i.Symbol)
			db.KlineDB.Collection(util.Md5Code(i.Id)).InsertMany(ctx, klines)
		})
	}
}

func getKline(symbol string) []model.Kline {
	url := fmt.Sprintf("https://stock.xueqiu.com/v5/stock/chart/kline.json?symbol=%s&begin=0&period=day&count=9999&type=before&indicator=kline,pe,pb,market_capital,agt,ggt,kdj,macd,boll,rsi,cci,balance", symbol)
	body, _ := util.XueQiuAPI(url)

	var data struct {
		Data struct {
			Column []string    `json:"column"`
			Item   [][]float64 `json:"item"`
		} `json:"data"`
	}
	util.UnmarshalJSON(body, &data)

	// read csv data
	var src strings.Builder
	src.WriteString(strings.Join(data.Data.Column, ","))

	for _, arr := range data.Data.Item {
		src.WriteByte('\n')
		for i := range arr {
			src.WriteString(strconv.FormatFloat(arr[i], 'f', 2, 32))

			if i != len(arr)-1 {
				src.WriteByte(',')
			}
		}
	}

	// phase csv data
	var klines []model.Kline
	if err := gocsv.Unmarshal(strings.NewReader(src.String()), &klines); err != nil {
		log.Warn().Msg(err.Error())
		return nil
	}

	return klines
}
