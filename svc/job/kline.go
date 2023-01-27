package job

import (
	"fmt"
	"fund/db"
	"fund/model"
	"fund/util"
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

	p := structx.NewPool()
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

	var kbm *model.KlineBitMap
	var zeroTime, _ = time.Parse("2006-01-02", "2012-01-01")

	// collection
	coll := db.KlineDB.Collection(util.Md5Code(id))
	coll.EnsureIndexes(ctx, []string{"code,time"}, nil)
	bulk := coll.Bulk()

	for i, k := range klines {
		k.Code = id
		k.Time /= 1000
		bulk.UpdateOne(bson.M{"code": k.Code, "time": k.Time}, bson.M{"$set": k})

		// bkm
		dur := time.Unix(k.Time, 0).Sub(zeroTime)
		// dates diff
		if dur > 0 {
			datesDiff := uint64(dur / (time.Hour * 24))

			if k.PctChg > 0 {
				kbm.PctChg.Add(datesDiff)
			}
			if k.MainNet > 0 {
				kbm.MainNet.Add(datesDiff)
			}
			if k.NetVolCN > 0 {
				kbm.HKHoldNet.Add(datesDiff)
			}
			if i > 0 && k.Vol > klines[i-1].Vol {
				kbm.VolChg.Add(datesDiff)
			}
		}
	}
	// run
	bulk.Run(ctx)
	coll.InsertMany(ctx, klines)

	db.LimitDB.Set(ctx, "kline:"+id, 1, time.Hour*24)
}
