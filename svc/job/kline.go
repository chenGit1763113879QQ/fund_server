package job

import (
	"fmt"
	"fund/cache"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

func InitKlines() {
	var stocks []struct {
		Id     string `bson:"_id"`
		Symbol string
	}
	db.Stock.Find(ctx, bson.M{"type": util.TYPE_STOCK}).All(&stocks)

	f := func(strs ...string) {
		symbol, id := strs[0], strs[1]

		// search cache
		if ok, _ := db.LimitDB.Exists(ctx, "kline:"+id).Result(); ok > 0 {
			return
		}
		// get kline
		klines := getKline(symbol, id)

		coll := db.TimeSeriesCollection(util.Md5Code(id))
		coll.EnsureIndexes(ctx, nil, []string{"meta.code"})
		coll.RemoveAll(ctx, bson.M{"meta.code": id})
		coll.InsertMany(ctx, klines)

		db.LimitDB.Set(ctx, "kline:"+id, 1, time.Hour*12)
	}

	p := util.NewPool(4)
	for _, i := range stocks {
		p.NewTask(f, i.Symbol, i.Id)
	}
	p.Wait()

	log.Info().Msg("init kline success.")
}

func getKline(symbol string, Id string) []*model.Kline {
	url := fmt.Sprintf("https://stock.xueqiu.com/v5/stock/chart/kline.json?symbol=%s&begin=1350000000000&period=day&count=9999&type=before&indicator=kline,pe,pb,market_capital,agt,ggt,kdj,macd,boll,rsi,cci,balance", symbol)
	body, _ := util.XueQiuAPI(url)

	var data struct {
		Data struct {
			Column []string    `json:"column"`
			Item   [][]float64 `json:"item"`
		} `json:"data"`
	}
	util.UnmarshalJSON(body, &data)

	klines := make([]*model.Kline, len(data.Data.Item))

	// reflect
	typeof := reflect.TypeOf(model.Kline{})

	// get csv tag
	tags := make([]string, typeof.NumField())
	for i := 0; i < typeof.NumField(); i++ {
		tags[i] = strings.Split(typeof.Field(i).Tag.Get("csv"), ",")[0]
	}

	for i, items := range data.Data.Item {
		// declare
		klines[i] = new(model.Kline)

		value := reflect.ValueOf(klines[i]).Elem()

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
		klines[i].Meta.Code = Id
		timeStr := time.Unix(klines[i].TimeStamp/1000, 0).Format("2006/01/02")
		klines[i].Time, _ = time.Parse("2006/01/02", timeStr)
	}

	return klines
}

func loadKlines() {
	log.Debug().Msg("kline start init")
	t, _ := time.Parse("2006/01/02", "2017/09/01")

	// run
	p := util.NewPool()
	for _, code := range getCNStocks() {
		p.NewTask(func(strs ...string) {
			id := strs[0]
			var data []*model.Kline

			// get kline
			db.KlineDB.Collection(util.Md5Code(id)).Aggregate(ctx, mongox.Pipeline().
				Match(bson.M{"meta.code": id, "time": bson.M{"$gt": t}}).
				Sort(bson.M{"time": 1}).Do()).All(&data)
			if data != nil {
				cache.KlineMap.Store(id, data)
			}
		}, code)
	}
	p.Wait()

	log.Debug().Msgf("init kline success, length:%d", cache.KlineMap.Len())
}
