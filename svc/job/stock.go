package job

import (
	"context"
	"fmt"
	"fund/cache"
	"fund/db"
	"fund/model"
	"fund/util"
	"strings"
	"sync"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	ctx = context.Background()

	Markets = []*model.Market{
		{Name: "CN", Type: "stock", Fs: "m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23,m:0+t:81+s:2048", Size: 5500},
		{Name: "CN", Type: "fund", Fs: "b:MK0021,b:MK0023", Size: 600},
		{Name: "CN", Type: "index", Fs: "m:1+s:2,m:0+t:5", Size: 400},

		{Name: "HK", Type: "stock", Fs: "m:128+t:3,m:128+t:4", Size: 1500},
		{Name: "HK", Type: "index", Fs: "i:100.HSI,i:100.HSCEI,i:124.HSTECH", Size: 10},

		{Name: "US", Type: "stock", Fs: "m:105,m:106,m:107", Size: 2500},
		{Name: "US", Type: "index", Fs: "i:100.NDX,i:100.DJIA,i:100.SPX", Size: 10},
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

	go func() {
		for {
			getMarketStatus()
			time.Sleep(time.Second)
		}
	}()
}

func GetTradeTime(code string) time.Time {
	splits := strings.Split(code, ".")
	if len(splits) == 2 {
		switch splits[1] {
		case "SH", "SZ", "BJ", "TI":
			return Markets[0].TradeTime
		case "HK":
			return Markets[3].TradeTime
		case "US":
			return Markets[5].TradeTime
		}
	}
	return time.Unix(0, 0)
}

func getRealStock(m *model.Market) {
	url := fmt.Sprintf("http://push2.eastmoney.com/api/qt/clist/get?po=1&fid=f20&pz=%d&np=1&fltt=2&pn=1&fs=%s&fields=", m.Size, m.Fs)
	stk := new(model.Stock)

	for {
		freq := m.Freq()
		if freq == 2 && m.Type == "stock" {
			log.Info().Msgf("updating stock[%s][%d]", m.Name, freq)
		}

		newUrl := url + strings.Join(stk.GetJsonFields(freq), ",")
		body, err := util.GetAndRead(newUrl)
		if err != nil {
			continue
		}

		var data []model.Stock
		util.UnmarshalJSON(body, &data, "data", "diff")

		bulk := db.Stock.Bulk()
		for i := range data {
			data[i].CalData(m)

			if data[i].Price > 0 {
				// update cache
				cache.Stock.Store(data[i].Id, data[i])

				// update db
				if freq >= 1 {
					bulk.UpdateId(data[i].Id, bson.M{"$set": data[i]})
				}

				// insert db
				if freq == 2 {
					db.Stock.InsertOne(ctx, data[i])
				}
			}
		}
		bulk.Run(ctx)

		updateMinute(data, m)

		if m.Type == "stock" && freq >= 1 {
			go getDistribution(m.Name)

			if m.Name == "CN" {
				go getIndustry(m)
				go getMainFlow()
				go getNorthMoney()
				go getAHCompare()
				go getMarketInfo()
			}
		}
		if m.Type == "stock" {
			Cond.Broadcast()
		}
		m.Incr()

		for !m.Status {
			time.Sleep(time.Millisecond * 100)
			m.ReSet()
		}
		time.Sleep(time.Millisecond * 300)
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
					"net": i.Net, "huge": i.MainHuge, "big": i.MainBig, "mid": i.MainMid,
					"small": i.MainSmall, "minutes": newTime.Minute()},
			)
		}
	}
	go bulk.Run(ctx)
}

func GetKline(items bson.M) {
	code := items["_id"].(string)
	cid, ok := items["cid"].(string)
	if !ok {
		return
	}

	// find cache
	res, _ := db.LimitDB.Exists(ctx, "kline:"+code).Result()
	if res > 0 {
		return
	}

	for _, p := range model.Params {
		url := fmt.Sprintf("http://push2his.eastmoney.com/api/qt/stock/kline/get?secid=%s&fields1=f1&fields2=f51,f52,f53,f54,f55,f56,f57,f59,f61&klt=%d&end=20500101&lmt=5000&fqt=1", cid, p.Params)
		body, _ := util.GetAndRead(url)

		var data []string
		util.UnmarshalJSON(body, &data, "data", "klines")

		// read csv
		df := dataframe.ReadCSV(strings.NewReader("time,open,close,high,low,vol,amount,pct_chg,tr\n" + strings.Join(data, "\n"))).
			Arrange(dataframe.Order{Colname: "time", Reverse: true})

		coll := db.KlineDB.Collection(util.Md5Code(code))

		// ensure index
		coll.EnsureIndexes(ctx, []string{"code,time"}, nil)

		bulk := coll.Bulk()
		docs := make([]map[string]interface{}, 0)

		// insert documents
		for _, i := range df.Maps() {
			i["code"] = code
			i["time"], _ = time.Parse(p.Format, i["time"].(string))

			docs = append(docs, i)
			bulk.UpdateOne(bson.M{"code": code, "time": i["time"]}, bson.M{"$set": i})
		}

		bulk.Run(ctx)
		coll.InsertMany(ctx, docs)
	}

	// update cache
	db.LimitDB.SetEX(ctx, "kline:"+code, "1", time.Hour*12)
}

func getAHCompare() {
	url := "https://xueqiu.com/service/v5/stock/ah/compare?page=1&order=desc&order_by=premium&size=2000"
	body, _ := util.GetAndRead(url)

	var data []model.AHCompare
	util.UnmarshalJSON(body, &data, "data", "items")

	bulk := db.Stock.Bulk()
	for _, i := range data {
		i.ParseId()
		bulk.UpdateId(i.CNCode, bson.M{"$set": bson.M{"AH": i}})
		bulk.UpdateId(i.HKCode, bson.M{"$set": bson.M{"AH": i}})
	}
	bulk.Run(ctx)
}
