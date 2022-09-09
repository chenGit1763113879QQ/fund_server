package job

import (
	"fund/cache"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"strings"
	"time"

	"github.com/go-gota/gota/dataframe"
	"go.mongodb.org/mongo-driver/bson"
)

func getIndustry(m *model.Market) {
	var data []model.Industry

	db.Stock.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"members": 1}).
		Lookup("stock", "members", "_id", "c").
		Project(bson.M{
			"c":          bson.M{"name": 1, "pct_chg": 1, "main_net": 1},
			"marketType": "$marketType",
			"type":       "$type",
			"close":      1,
			"pct_chg":    bson.M{"$avg": "$c.pct_chg"},
			"main_net":   bson.M{"$sum": "$c.main_net"},
			"vol":        bson.M{"$sum": "$c.vol"},
			"tr":         bson.M{"$avg": "$c.tr"},
			"amount":     bson.M{"$sum": "$c.amount"},
			"mc":         bson.M{"$sum": "$c.mc"},
			"fmc":        bson.M{"$sum": "$c.fmc"},
			"pe_ttm":     bson.M{"$avg": "$c.pe_ttm"},
			"pb":         bson.M{"$avg": "$c.pb"},
			"pct_year":   bson.M{"$avg": "$c.pct_year"},
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
		i.Price = i.Close * (1 + float64(i.PctChg)/100)
		bulk.UpdateId(i.Id, bson.M{"$set": i})

		// minute data
		if m.Status {
			minBulk.UpsertId(
				bson.M{"code": i.Id, "time": newTime.Unix()},
				bson.M{"price": i.Price, "pct_chg": i.PctChg, "vol": i.Vol, "minutes": newTime.Minute()},
			)
		}
	}
	bulk.Run(ctx)
	minBulk.Run(ctx)
}

func getDistribution(market uint8) {
	var data []struct {
		Count int64 `bson:"count"`
	}
	var cn uint8 = util.MARKET_CN
	var stock uint8 = util.TYPE_STOCK

	db.Stock.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"marketType": market, "type": stock, "price": bson.M{"$gt": 0}}).
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

	if market == cn {
		nums[0], _ = db.Stock.Find(ctx, bson.M{"marketType": cn, "type": stock, "pct_chg": bson.M{"$lt": -9.8}}).Count()
		label[0] = "跌停"
		nums[10], _ = db.Stock.Find(ctx, bson.M{"marketType": cn, "type": stock, "pct_chg": bson.M{"$gt": 9.8}}).Count()
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
	codes := "SH000001,SZ399001,SZ399006,SH000688,HKHSI,HKHSCCI,HKHSCEI,.DJI,.IXIC,.INX,ICS30"
	body, _ := util.GetAndRead("https://xueqiu.com/service/v5/stock/batch/quote?symbol=" + codes)

	var indexes []model.Index
	util.UnmarshalJSON(body, &indexes, "data", "items")

	for _, i := range indexes {
		for _, p := range Markets {
			if p.StrMarket == i.Market.Region {
				i.Stock.CalData(p)
				// status
				p.StatusName = i.Market.StatusName
				p.Status = p.StatusName == "交易中" || p.StatusName == "集合竞价"

				// tradeTime
				cst, _ := time.LoadLocation(i.Market.TimeZone)
				p.TradeTime = time.Unix(i.Stock.Time/1000, 0).In(cst)

				i.Stock.MarketType = p.Market
				i.Stock.Type = util.TYPE_INDEX

				db.Stock.InsertOne(ctx, i.Stock)
				db.Stock.UpdateId(ctx, i.Stock.Id, i)
				break
			}
		}
	}
}
