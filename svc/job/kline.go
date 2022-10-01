package job

import (
	"fmt"
	"fund/cache"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"strings"
	"time"

	"github.com/bytedance/sonic"
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

	funcDaily := func(strs ...string) {
		symbol, id := strs[0], strs[1]
		// find cache
		if ok, _ := db.LimitDB.Exists(ctx, "kline:day:"+id).Result(); ok > 0 {
			return
		}

		// get kline
		klines := getKline(symbol, id)
		if klines == nil {
			return
		}

		coll := db.KlineDB.Collection(util.Md5Code(id))
		coll.EnsureIndexes(ctx, []string{"code,time"}, nil)
		coll.Remove(ctx, bson.M{"code": id})
		coll.InsertMany(ctx, klines)

		db.LimitDB.Set(ctx, "kline:day:"+id, 1, time.Hour*12)
	}

	funcMinute := func(strs ...string) {
		id := strs[0]
		// find cache
		if ok, _ := db.LimitDB.Exists(ctx, "kline:1m:"+id).Result(); ok > 0 {
			return
		}

		// get kline
		klines := getMinuteKline(id)
		if klines == nil {
			return
		}

		coll := db.MKlineDB.Collection(util.Md5Code(id))
		coll.EnsureIndexes(ctx, []string{"code,time"}, nil)
		coll.Remove(ctx, bson.M{"code": id})
		coll.InsertMany(ctx, klines)

		db.LimitDB.Set(ctx, "kline:1m:"+id, 1, time.Hour*6)
	}

	p := util.NewPool(2)
	for _, i := range stocks {
		p.NewTask(funcDaily, i.Symbol, i.Id)
		p.NewTask(funcMinute, i.Id)
	}
	p.Wait()

	log.Info().Msg("init kline success.")
}

func getKline(symbol string, Id string) []*model.Kline {
	ts := time.Now().UnixMilli()
	url := fmt.Sprintf("https://stock.xueqiu.com/v5/stock/chart/kline.json?symbol=%s&begin=%d&period=day&count=-4500&indicator=kline,pe,pb,ps,pcf,market_capital,agt,ggt,macd,boll,balance", symbol, ts)
	body, _ := util.XueQiuAPI(url)

	// get node
	node, _ := sonic.Get(body, "data")
	raw, _ := node.Raw()

	var data struct {
		Column []string `json:"column"`
		Item   [][]any  `json:"item"`
	}
	// unmarshal
	if err := util.UnmarshalJSON([]byte(raw), &data); err != nil {
		log.Err(err)
		return nil
	}

	// struct to map
	srcMap := make([]map[string]any, len(data.Item))
	for i, item := range data.Item {
		srcMap[i] = map[string]any{}

		for c, col := range data.Column {
			srcMap[i][col] = item[c]
		}
	}

	var klines []*model.Kline
	mapstructure.Decode(srcMap, klines)

	// set code and time
	layout := "2006/01/02"
	for _, k := range klines {
		k.Code = Id
		k.Time, _ = time.Parse(layout, time.UnixMilli(k.TimeStamp).Format(layout))
	}
	return klines
}

func getMinuteKline(id string) []*model.MinuteKline {
	before, after, _ := strings.Cut(id, ".")
	if after == "SS" {
		id = before + ".SH"
	}

	url := fmt.Sprintf("https://api-ddc-wscn.xuangubao.cn/market/kline?tick_count=10000&prod_code=%s&fields=tick_at,open_px,close_px,avg_px", id)
	body, _ := util.GetAndRead(url)

	var data struct {
		Column []string
		Item   [][]float64
	}

	// get node
	node, _ := sonic.Get(body, "data", "candle", id, "lines")
	itemStr, _ := node.Raw()

	node, _ = sonic.Get(body, "data", "fields")
	colStr, _ := node.Raw()

	// unmarshal
	if err := util.UnmarshalJSON([]byte(itemStr), &data.Item); err != nil {
		log.Err(err)
		return nil
	}
	if err := util.UnmarshalJSON([]byte(colStr), &data.Column); err != nil {
		log.Err(err)
		return nil
	}

	// data maps
	dataMaps := make([]map[string][]float64, 0)

	for _, item := range data.Item {
		maps := map[string][]float64{}

		for j, col := range data.Column {
			_, ok := maps[col]
			if ok {
				maps[col] = append(maps[col], item[j])
			}
		}
		dataMaps = append(dataMaps, maps)
	}

	var klines []*model.MinuteKline
	mapstructure.Decode(dataMaps, klines)

	fmt.Println(klines[0])

	return nil
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
				Match(bson.M{"code": id, "time": bson.M{"$gt": t}}).
				Sort(bson.M{"time": 1}).Do()).All(&data)

			cache.KlineMap.Store(id, data)
		}, code)
	}
	p.Wait()

	log.Debug().Msgf("init kline[%d] success", cache.KlineMap.Len())
}
