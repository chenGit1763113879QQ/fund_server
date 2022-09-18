package job

import (
	"fmt"
	"fund/db"
	"fund/model"
	"fund/util"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

func getRealStock(m *model.Market) {
	url := fmt.Sprintf("https://xueqiu.com/service/v5/stock/screener/quote/list?size=5000&order_by=amount&type=%s", m.StrType)

	for {
		freq := m.Freq()

		if freq == 2 {
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

		go bulk.Run(ctx)
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
					"main_net": i.MainNet},
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
	db.Stock.Find(ctx, bson.M{"type": util.TYPE_STOCK}).All(&stocks)

	for _, i := range stocks {
		// search cache
		if ok, _ := db.LimitDB.Exists(ctx, "kline:"+i.Id).Result(); ok > 0 {
			continue
		}

		// get kline
		klines := getKline(i.Symbol, i.Id)

		coll := db.KlineDB.Collection(util.Md5Code(i.Id))
		coll.EnsureIndexes(ctx, []string{"code,time"}, nil)
		coll.RemoveAll(ctx, bson.M{"code": i.Id})
		coll.InsertMany(ctx, klines)

		db.LimitDB.Set(ctx, "kline:"+i.Id, 1, time.Hour*12)
	}
	log.Info().Msg("init kline success.")
}

func getKline(symbol string, Id string) []model.Kline {
	url := fmt.Sprintf("https://stock.xueqiu.com/v5/stock/chart/kline.json?symbol=%s&begin=1350000000000&period=day&count=9999&type=before&indicator=kline,pe,pb,market_capital,agt,ggt,kdj,macd,boll,rsi,cci,balance", symbol)
	body, _ := util.XueQiuAPI(url)

	var data struct {
		Data struct {
			Column []string    `json:"column"`
			Item   [][]float64 `json:"item"`
		} `json:"data"`
	}
	util.UnmarshalJSON(body, &data)

	klines := make([]model.Kline, len(data.Data.Item))

	// reflect
	typeof := reflect.TypeOf(model.Kline{})

	// get csv tag
	tags := make([]string, typeof.NumField())
	for i := 0; i < typeof.NumField(); i++ {
		tags[i] = strings.Split(typeof.Field(i).Tag.Get("csv"), ",")[0]
	}

	for i, items := range data.Data.Item {
		value := reflect.ValueOf(&klines[i]).Elem()

		for colI, col := range data.Data.Column {
			for tagI, tag := range tags {
				if tag == col {
					// null number
					if items[colI] < 0.0001 || items[colI] > math.Pow(10, 16) {
						break
					}

					// set value
					switch value.Field(tagI).Kind() {
					case reflect.Float64:
						value.Field(tagI).SetFloat(items[colI])

					case reflect.Int64:
						value.Field(tagI).SetInt(int64(items[colI]))
					}
					break
				}
			}
		}
	}

	// set
	for i := range klines {
		klines[i].Code = Id
		timeStr := time.Unix(klines[i].TimeStamp/1000, 0).Format("2006/01/02")
		klines[i].Time, _ = time.Parse("2006/01/02", timeStr)
	}

	return klines
}
