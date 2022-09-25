package job

import (
	"fmt"
	"fund/cache"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"time"

	"github.com/bytedance/sonic"
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
		symbol, id := strs[0], strs[1]
		// find cache
		if ok, _ := db.LimitDB.Exists(ctx, "kline:5m:"+id).Result(); ok > 0 {
			return
		}

		// get kline
		klines := getMinuteKline(symbol, id)
		if klines == nil {
			return
		}

		coll := db.MKlineDB.Collection(util.Md5Code(id))
		coll.EnsureIndexes(ctx, []string{"code,time"}, nil)
		coll.Remove(ctx, bson.M{"code": id})
		coll.InsertMany(ctx, klines)

		db.LimitDB.Set(ctx, "kline:5m:"+id, 1, time.Hour)
	}

	p := util.NewPool(2)
	for _, i := range stocks {
		p.NewTask(funcDaily, i.Symbol, i.Id)
		p.NewTask(funcMinute, i.Symbol, i.Id)
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

	// decompress and unmarshal
	var klines []*model.Kline
	if err := util.DeCompressJSON([]byte(raw), &klines); err != nil {
		return nil
	}

	// set
	for _, k := range klines {
		k.Code = Id
		timeStr := time.UnixMilli(k.TimeStamp).Format("2006/01/02")
		k.Time, _ = time.Parse("2006/01/02", timeStr)
	}
	return klines
}

func getMinuteKline(symbol string, Id string) []*model.Kline {
	ts := time.Now().UnixMilli()

	url := fmt.Sprintf("https://stock.xueqiu.com/v5/stock/chart/kline.json?symbol=%s&begin=%d&period=5m&count=-5000&indicator=kline", symbol, ts)
	body, _ := util.XueQiuAPI(url)

	// get node
	node, _ := sonic.Get(body, "data")
	raw, _ := node.Raw()

	// decompress and unmarshal
	var klines []*model.Kline
	if err := util.DeCompressJSON([]byte(raw), &klines); err != nil {
		return nil
	}

	// set
	for _, k := range klines {
		k.Code = Id
		timeStr := time.UnixMilli(k.TimeStamp).Format("2006/01/02 15:04")
		k.Time, _ = time.Parse("2006/01/02 15:04", timeStr)
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
				Match(bson.M{"code": id, "time": bson.M{"$gt": t}}).
				Sort(bson.M{"time": 1}).Do()).All(&data)
			if data != nil {
				cache.KlineMap.Store(id, data)
			}
		}, code)
	}
	p.Wait()

	log.Debug().Msgf("init kline[%d] success", cache.KlineMap.Len())
}
