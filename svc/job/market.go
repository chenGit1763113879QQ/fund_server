package job

import (
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"strings"
	"time"

	"github.com/go-gota/gota/dataframe"
	"go.mongodb.org/mongo-driver/bson"
)

func getDistribution(m *model.Market) {
	var data []struct {
		Count int64 `bson:"count"`
	}

	db.Stock.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"marketType": m.Market, "type": m.Type, "price": bson.M{"$gt": 0}}).
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

	if m.Market == util.CN {
		label[0] = "跌停"
		label[10] = "涨停"
	}
	db.Numbers.Store(m.Market, bson.M{"label": label, "value": nums})
}

func getNorthMoney() {
	url := "http://push2.eastmoney.com/api/qt/kamt.rtmin/get?fields1=f1,f3&fields2=f51,f52,f54"
	body, _ := util.GetAndRead(url)

	var data []string
	util.UnmarshalJSON(body, &data, "data", "s2n")

	df := dataframe.ReadCSV(strings.NewReader("time,hgt,sgt\n" + strings.Join(data, "\n")))
	db.NorthMoney = df.Maps()
}

func getMainFlow() {
	url := "http://push2.eastmoney.com/api/qt/stock/fflow/kline/get?lmt=0&klt=1&fields1=f1&fields2=f51,f52&secid=1.000001&secid2=0.399001"
	body, _ := util.GetAndRead(url)

	var data []string
	util.UnmarshalJSON(body, &data, "data", "klines")

	df := dataframe.ReadCSV(strings.NewReader("time,value\n" + strings.Join(data, "\n")))
	db.MainFlow = df.Maps()
}

func getMarketStatus() {
	codes := "SH000001,SZ399001,SZ399006,SH000688,HKHSI,HKHSCCI,HKHSCEI,.DJI,.IXIC,.INX,ICS30"
	body, _ := util.GetAndRead("https://xueqiu.com/service/v5/stock/batch/quote?symbol=" + codes)

	var indexes []*model.Index
	util.UnmarshalJSON(body, &indexes, "data", "items")

	for _, i := range indexes {
		for _, p := range Markets {
			if string(p.Market) == i.Market.Region {
				i.Stock.CalData(p)
				// status
				p.StatusName = i.Market.StatusName
				p.Status = p.StatusName == "交易中" || p.StatusName == "集合竞价"

				// tradeTime
				cst, _ := time.LoadLocation(i.Market.TimeZone)
				p.TradeTime = time.UnixMilli(i.Stock.Time).In(cst)

				i.Stock.Type = util.INDEX

				db.Stock.UpdateId(ctx, i.Stock.Id, bson.M{"$set": i.Stock})
				db.Stock.InsertOne(ctx, i.Stock)
			}
		}
	}
}

func GetMarket(code string) *model.Market {
	splits := strings.Split(code, ".")
	if len(splits) == 2 {
		switch splits[1] {
		case "CN", "SH", "SZ", "BJ":
			return Markets[0]
		case "HK":
			return Markets[1]
		case "US":
			return Markets[2]
		}
	}
	return nil
}
