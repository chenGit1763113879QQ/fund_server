package job

import (
	"fmt"
	"fund/db"
	"fund/model"
	"fund/util"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/xgzlucario/structx"
	"go.mongodb.org/mongo-driver/bson"
)

func initKline() {
	var stocks []struct {
		Id     string `bson:"_id"`
		Symbol string
	}
	db.Stock.Find(ctx, bson.M{"type": util.STOCK}).All(&stocks)

	p := structx.NewPool[string](4)
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

	url := fmt.Sprintf("https://stock.xueqiu.com/v5/stock/chart/kline.json?symbol=%s&type=before&begin=%d&period=day&count=-4500&indicator=kline,pe,pb,ps,pcf,agt,ggt,boll,balance", symbol, time.Now().UnixMilli())
	body, _ := util.XueQiuAPI(url)

	// unmarshal
	var data struct {
		Column []string `json:"column"`
		Item   [][]any  `json:"item"`
	}
	util.UnmarshalJSON(body, &data, "data")

	var klines []*model.Kline
	if err := util.DecodeJSONItems(data.Column, data.Item, &klines); err != nil {
		return
	}

	// collection
	coll := db.KlineDB.Collection(util.Md5Code(id))
	coll.EnsureIndexes(ctx, []string{"code,time"}, nil)
	bulk := coll.Bulk()

	for _, k := range klines {
		k.Code = id
		k.Time /= 1000
		bulk.UpdateOne(bson.M{"code": k.Code, "time": k.Time}, bson.M{"$set": k})
	}
	// run
	bulk.Run(ctx)
	coll.InsertMany(ctx, klines)

	db.LimitDB.Set(ctx, "kline:day:"+id, 1, time.Hour*24)

	// winner_rate
	getWinRate(id)
}

func getMinuteKline(strs ...string) {
	id := strs[0]
	if !util.IsCNStock(id) {
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

	url := "https://api-ddc-wscn.xuangubao.cn/market/kline?tick_count=10000&fields=tick_at,close_px&prod_code=" + symbol
	body, _ := util.GetAndRead(url)

	var data [][]float64
	util.UnmarshalJSON(body, &data, "data", "candle", id, "lines")

	// coll
	db.Minute.RemoveAll(ctx, bson.M{"code": id})
	bulk := db.Minute.Bulk()

	price := make([]float64, 0)
	var t, tradeDate time.Time

	for _, item := range data {
		t = time.Unix(int64(item[1]), 0)

		if !tradeDate.IsZero() && t.Day() > tradeDate.Day() {
			bulk.InsertOne(bson.M{
				"code": id,
				// "price":      oneness(price),
				"trade_date": t.Format("2006/01/02"),
			})
			price = make([]float64, 0)
		}

		tradeDate = t
		price = append(price, item[0])
	}

	bulk.Run(ctx)
	db.LimitDB.Set(ctx, "kline:1m:"+id, 1, time.Hour*24)
}

func getWinRate(id string) {
	return
	if !util.IsCNStock(id) {
		return
	}
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
		"cyq_perf",
		bson.M{"ts_code": id}, "trade_date,weight_avg,winner_rate", &data,
	)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	bulk := db.KlineDB.Collection(util.Md5Code(id)).Bulk()

	// save db
	for _, i := range data {
		t, _ := time.Parse("20060102", i.TradeDate)
		bulk.UpdateOne(bson.M{"code": id, "time": t.Unix()}, bson.M{"$set": i})
	}
	bulk.Run(ctx)

	db.LimitDB.Set(ctx, "winner_rate:"+id, 1, time.Hour*24)
}
