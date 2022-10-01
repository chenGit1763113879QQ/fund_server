package job

import (
	"fmt"
	"fund/db"
	"fund/model"
	"fund/util"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

func InitKlines() {
	var stocks []struct {
		Id     string `bson:"_id"`
		Symbol string
	}
	db.Stock.Find(ctx, bson.M{"type": util.TYPE_STOCK}).All(&stocks)

	p := util.NewPool(4)
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

	var data struct {
		Column []string `json:"column"`
		Item   [][]any  `json:"item"`
	}
	util.UnmarshalJSON(body, &data, "data")

	// decode map
	srcMap := make([]map[string]any, len(data.Item))
	for i, item := range data.Item {
		srcMap[i] = map[string]any{}

		for c, col := range data.Column {
			srcMap[i][col] = item[c]
		}
	}

	var klines []*model.Kline
	mapstructure.Decode(srcMap, &klines)
	if klines == nil {
		return
	}

	for _, k := range klines {
		k.Code = id
		k.Time = time.UnixMilli(k.TimeStamp)
	}

	// save
	coll := db.KlineDB.Collection(util.Md5Code(id))
	coll.EnsureIndexes(ctx, []string{"code,time"}, nil)
	coll.RemoveAll(ctx, bson.M{"code": id})
	coll.InsertMany(ctx, klines)

	db.LimitDB.Set(ctx, "kline:day:"+id, 1, time.Hour*12)
}

func getMinuteKline(strs ...string) {
	id := strs[0]

	var symbol string
	before, after, _ := strings.Cut(id, ".")
	switch after {
	case "SH":
		symbol = before + ".SS"
	case "SZ":
		symbol = id
	default:
		return
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
	util.UnmarshalJSON(body, &data.Column, "data", "fields")
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
	db.LimitDB.Set(ctx, "kline:1m:"+id, 1, time.Hour*6)
}
