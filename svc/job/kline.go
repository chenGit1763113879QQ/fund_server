package job

import (
	"fmt"
	"fund/db"
	"fund/model"
	"fund/util"
	"runtime"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sourcegraph/conc/pool"
	"go.mongodb.org/mongo-driver/bson"
)

func initKline() {
	var stocks []struct {
		Id     string `bson:"_id"`
		Symbol string
	}
	db.Stock.Find(ctx, bson.M{"type": util.STOCK}).All(&stocks)

	p := pool.New().WithMaxGoroutines(runtime.NumCPU())
	for _, i := range stocks {
		s, id := i.Symbol, i.Id
		p.Go(func() {
			getKline(s, id)
		})
	}
	p.Wait()
	log.Info().Msgf("init kline[%d] success", len(stocks))
}

func getKline(symbol, id string) {
	// find cache
	// if ok, _ := db.LimitDB.Exists(ctx, "kline:"+id).Result(); ok > 0 {
	// 	return
	// }

	url := fmt.Sprintf("https://stock.xueqiu.com/v5/stock/chart/kline.json?symbol=%s&type=before&begin=%d&period=day&count=-4500&indicator=kline,pe,pb,ps,pcf,agt,ggt,boll,balance", symbol, time.Now().UnixMilli())
	body, _ := util.XueQiuAPI(url)

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
		k.Time /= 1000
	}

	const days = 30

	for i := days; i < len(klines); i++ {
		kline := klines[i]

		// 提取数据范围
		ranges := klines[i-days : i]

		for j, tmp := range ranges {
			if tmp.PctChg > 0 {
				kline.Stat.PctChg |= 1 << j
			}
			if tmp.MainNet > 0 {
				kline.Stat.MainNet |= 1 << j
			}
			if tmp.NetVolCN > 0 {
				kline.Stat.HKHoldNet |= 1 << j
			}
		}
	}

	for _, k := range klines {
		k.Code = id
		bulk.UpdateOne(bson.M{"code": k.Code, "time": k.Time}, bson.M{"$set": k})
	}

	// run
	bulk.Run(ctx)
	coll.InsertMany(ctx, klines)

	fmt.Println(id)

	// db.LimitDB.Set(ctx, "kline:"+id, 1, time.Hour*24)
}
