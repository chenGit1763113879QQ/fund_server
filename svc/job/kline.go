package job

import (
	"fmt"
	"fund/db"
	"fund/model"
	"fund/util"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

var count int32 = 195

func init() {
	// start limit count
	go func() {
		for {
			time.Sleep(time.Minute)
			atomic.SwapInt32(&count, 195)
		}
	}()
}

func InitKlines() {
	var stocks []struct {
		Id     string `bson:"_id"`
		Symbol string
	}
	db.Stock.Find(ctx, bson.M{"type": util.TYPE_STOCK}).All(&stocks)

	p := util.NewPool(8)
	// kline
	for _, i := range stocks {
		p.NewTask(getKline, i.Symbol, i.Id)
		p.NewTask(getMinuteKline, i.Id)
	}
	p.Wait()
	log.Info().Msgf("init kline[%d] success", len(stocks))
}

func getKline(strs ...string) {
	symbol, id := strs[0], strs[1]
	// find cache
	if ok, _ := db.LimitDB.Exists(ctx, "kline:day:"+id).Result(); ok > 0 {
		return
	}

	url := fmt.Sprintf("https://stock.xueqiu.com/v5/stock/chart/kline.json?symbol=%s&type=before&begin=%d&period=day&count=-4500&indicator=kline,pe,pb,ps,pcf,market_capital,agt,ggt,macd,boll,balance", symbol, time.Now().UnixMilli())
	body, _ := util.XueQiuAPI(url)

	// unmarshal
	var data struct {
		Column []string `json:"column"`
		Item   [][]any  `json:"item"`
	}
	util.UnmarshalJSON(body, &data, "data")

	var klines []*model.Kline
	// decode
	util.DecodeJSONItems(data.Column, data.Item, &klines)
	if klines == nil {
		return
	}

	// set code and time
	layout := "2006/01/02"
	for _, k := range klines {
		k.Code = id
		k.Time, _ = time.Parse(layout, time.UnixMilli(k.TimeStamp).Format(layout))
	}

	// save
	coll := db.KlineDB.Collection(util.Md5Code(id))
	coll.EnsureIndexes(ctx, []string{"code,time"}, nil)
	coll.RemoveAll(ctx, bson.M{"code": id})
	coll.InsertMany(ctx, klines)

	db.LimitDB.Set(ctx, "kline:day:"+id, 1, time.Hour*24)

	// winner_rate
	go getWinRate(id)
}

func getMinuteKline(strs ...string) {
	id := strs[0]
	// is cn stock
	if !strings.Contains(id, ".SH") && !strings.Contains(id, ".SZ") {
		return
	}

	symbol := id
	if strings.Contains(id, ".SH") {
		symbol = strings.ReplaceAll(id, ".SH", ".SS")
	}

	// find cache
	if ok, _ := db.LimitDB.Exists(ctx, "kline:1m:"+id).Result(); ok > 0 {
		return
	}

	url := fmt.Sprintf("https://api-ddc-wscn.xuangubao.cn/market/kline?tick_count=10000&prod_code=%s&fields=tick_at,open_px,close_px,avg_px", symbol)
	body, _ := util.GetAndRead(url)

	var data struct {
		Column []string
		Item   [][]float64
	}

	// unmarshal
	if err := util.UnmarshalJSON(body, &data.Column, "data", "fields"); err != nil {
		log.Error().Msg(err.Error())
	}
	util.UnmarshalJSON(body, &data.Item, "data", "candle", id, "lines")

	// coll
	db.Minute.RemoveAll(ctx, bson.M{"code": id})
	bulk := db.Minute.Bulk()

	kline := &model.MinuteKline{
		Code:  id,
		Time:  make([]int64, 0),
		Open:  make([]float64, 0),
		Close: make([]float64, 0),
		Avg:   make([]float64, 0),
	}

	for _, item := range data.Item {
		t := time.Unix(int64(item[2]), 0)

		if !kline.TradeDate.IsZero() && t.Day() > kline.TradeDate.Day() {
			// 归一化
			factor := kline.Open[0]
			oneness(kline.Close, factor)
			oneness(kline.Open, factor)
			oneness(kline.Avg, factor)
			bulk.InsertOne(kline)

			kline = &model.MinuteKline{
				Code:  id,
				Time:  make([]int64, 0),
				Open:  make([]float64, 0),
				Close: make([]float64, 0),
				Avg:   make([]float64, 0),
			}
		}
		kline.TradeDate = t
		kline.Open = append(kline.Open, item[0])
		kline.Close = append(kline.Close, item[1])
		kline.Time = append(kline.Time, int64(item[2]))
		kline.Avg = append(kline.Avg, item[3])
	}

	bulk.Run(ctx)
	db.LimitDB.Set(ctx, "kline:1m:"+id, 1, time.Hour*24)
}

func getWinRate(id string) {
	// is cn stock
	if !strings.Contains(id, ".SH") && !strings.Contains(id, ".SZ") {
		return
	}

	// count
	for atomic.LoadInt32(&count) < 1 {
	}
	atomic.AddInt32(&count, -1)

	// find cache
	if ok, _ := db.LimitDB.Exists(ctx, "winner_rate:"+id).Result(); ok > 0 {
		return
	}

	var data []*struct {
		TradeDate  string  `bson:"-" mapstructure:"trade_date"`
		WeightAvg  float64 `bson:"weight_avg" mapstructure:"weight_avg"`
		WinnerRate float64 `bson:"winner_rate" mapstructure:"winner_rate"`
	}
	err := util.TushareApi(
		"cyq_perf", bson.M{"ts_code": id}, "trade_date,weight_avg,winner_rate", &data,
	)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	bulk := db.KlineDB.Collection(util.Md5Code(id)).Bulk()

	// save db
	for _, i := range data {
		t, _ := time.Parse("20060102", i.TradeDate)
		bulk.UpdateOne(bson.M{"code": id, "time": t}, bson.M{"$set": i})
	}
	bulk.Run(ctx)

	db.LimitDB.Set(ctx, "winner_rate:"+id, 1, time.Hour*24)
}
