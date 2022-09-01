package job

import (
	"fund/cache"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/go-gota/gota/dataframe"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

const StkHost = "https://flash-api.xuangubao.cn/api"

// update industry
func getIndustry(m *model.Market) {
	var data []model.Industry

	db.Stock.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"type": bson.M{"$in": bson.A{"I1", "I2", "C"}}}).
		Lookup("stock", "members", "_id", "c").
		Project(bson.M{
			"c":          bson.M{"name": 1, "pct_chg": 1, "main_net": 1},
			"marketType": "CN",
			"type":       "$type",
			"high":       "$high",
			"low":        "$low",
			"close":      1,
			"pct_chg":    bson.M{"$avg": "$c.pct_chg"},

			"main_huge":  bson.M{"$sum": "$c.main_huge"},
			"main_big":   bson.M{"$sum": "$c.main_big"},
			"main_mid":   bson.M{"$sum": "$c.main_mid"},
			"main_small": bson.M{"$sum": "$c.main_small"},
			"main_net":   bson.M{"$add": bson.A{"$main_huge", "$main_big"}},

			"net":      bson.M{"$sum": "$c.net"},
			"vol":      bson.M{"$sum": "$c.vol"},
			"tr":       bson.M{"$avg": "$c.tr"},
			"amount":   bson.M{"$sum": "$c.amount"},
			"mc":       bson.M{"$sum": "$c.mc"},
			"fmc":      bson.M{"$sum": "$c.fmc"},
			"pe_ttm":   bson.M{"$avg": "$c.pe_ttm"},
			"pb":       bson.M{"$avg": "$c.pb"},
			"wb":       bson.M{"$avg": "$c.wb"},
			"pct_year": bson.M{"$avg": "$c.pct_year"},
		}).Do()).All(&data)

	bulk := db.Stock.Bulk()

	tradeTime := m.TradeTime.Format("2006/01/02 15:04")
	date := strings.Split(tradeTime, " ")[0]

	newTime, _ := time.Parse("2006/01/02 15:04", tradeTime)

	minBulk := db.MinuteDB.Collection(date).Bulk()

	for _, i := range data {
		i.PctLeader.PctChg = -100
		i.MainNetLeader.MainNet = -999999

		// leader stock
		for _, stk := range i.ConnList {
			if stk.PctChg > i.PctLeader.PctChg {
				i.PctLeader = stk
			}
			if stk.MainNet > i.MainNetLeader.MainNet {
				i.MainNetLeader = stk
			}
		}
		i.ConnList = nil

		// price
		i.Price = i.Close * (1 + i.PctChg/100)
		if i.Price > i.High {
			i.High = i.Price
		}
		if i.Price < i.Low {
			i.Low = i.Price
		}
		bulk.UpdateId(i.Id, bson.M{"$set": i})

		// minute data
		if m.Status {
			minBulk.UpsertId(
				bson.M{"code": i.Id, "time": newTime.Unix()},
				bson.M{"price": i.Price, "pct_chg": i.PctChg, "vol": i.Vol, "net": i.Net, "huge": i.MainHuge,
					"big": i.MainBig, "mid": i.MainMid, "small": i.MainSmall, "minutes": newTime.Minute()},
			)
		}
	}
	bulk.Run(ctx)
	minBulk.Run(ctx)
}

// get market distribution
func getDistribution(market string) {
	return

	var data []struct {
		Count int64 `bson:"count"`
	}
	db.Stock.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"marketType": market, "type": "stock", "price": bson.M{"$gt": 0}}).
		Bucket(
			"$pct_chg",
			bson.A{-99, -10, -7, -5, -3, -0.0001, 0.0001, 3, 5, 7, 10, 999},
			"Other",
			bson.M{"count": bson.M{"$sum": 1}}).
		Do()).All(&data)

	label := []string{"<10", "<7", "7-5", "5-3", "3-0", "0", "0-3", "3-5", "5-7", ">7", ">10"}
	nums := make([]int64, 11)
	for i := range data {
		nums[i] = data[i].Count
	}
	if market == "CN" {
		nums[0], _ = db.Stock.Find(ctx, bson.M{"marketType": "CN", "type": "stock", "pct_chg": bson.M{"$lt": -9.8}}).Count()
		label[0] = "跌停"
		nums[10], _ = db.Stock.Find(ctx, bson.M{"marketType": "CN", "type": "stock", "pct_chg": bson.M{"$gt": 9.8}}).Count()
		label[10] = "涨停"
	}
	cache.Numbers.Store(market, bson.M{"label": label, "value": nums})
}

func getNorthMoney() {
	url := "http://push2.eastmoney.com/api/qt/kamt.rtmin/get?fields1=f1,f3&fields2=f51,f52,f54"
	body, _ := util.GetAndRead(url)

	var data []string
	util.UnmarshalJSON(body, &data, "data", "s2n")

	df := dataframe.ReadCSV(strings.NewReader("time,hgt,sgt\n" + strings.Join(data, "\n")))
	cache.NorthMoney = df.Maps()
}

func getMainFlow() {
	url := "http://push2.eastmoney.com/api/qt/stock/fflow/kline/get?lmt=0&klt=1&fields1=f1&fields2=f51,f52&secid=1.000001&secid2=0.399001"
	body, _ := util.GetAndRead(url)

	var data []string
	util.UnmarshalJSON(body, &data, "data", "klines")

	df := dataframe.ReadCSV(strings.NewReader("time,value\n" + strings.Join(data, "\n")))
	cache.MainFlow = df.Maps()
}

func getMarketStatus() {
	body, _ := util.GetAndRead("https://xueqiu.com/service/v5/stock/batch/quote?symbol=SH000001,HKHSI,.IXIC")

	items, _ := sonic.Get(body, "data", "items")
	for i := 0; i < 3; i++ {
		item := items.Index(i)

		name, _ := item.Get("market").Get("region").String()

		for _, p := range Markets {
			if p.Name == name {
				p.StatusName, _ = item.Get("market").Get("status").String()
				p.Status = p.StatusName == "交易中" || p.StatusName == "集合竞价"

				timeZone, _ := item.Get("market").Get("time_zone").String()
				cst, _ := time.LoadLocation(timeZone)

				ts, _ := item.Get("quote").Get("timestamp").Int64()
				p.TradeTime = time.Unix(ts/1000, 0).In(cst)

				// valid
				if p.TradeTime.IsZero() {
					log.Error().Str("error", "trade_time is zero")
					getMarketStatus()
				}
			}
		}
	}
}

func getMarketInfo() {
	url := StkHost + "/market_indicator/line?fields=rise_count,fall_count,yesterday_limit_up_avg_pcp,limit_up_count,limit_down_count,limit_up_broken_count,limit_up_broken_ratio,market_temperature"
	body, _ := util.GetAndRead(url)

	util.UnmarshalJSON(body, &cache.MarketHot, "data")
}
