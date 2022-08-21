package pro

import (
	"context"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"fund/util/pool"
	"math"

	"go.mongodb.org/mongo-driver/bson"
)

type Stockx struct {
	Id     string `bson:"_id"`
	Market string `bson:"marketType"`
	Name   string
	Mc     float64
}

var (
	ctx   = context.Background()
	items = initStock()
)

func init() {
	initKline()
}

// init stocks
func initStock() []Stockx {
	var data []Stockx
	db.Stock.Find(ctx, bson.M{
		"type": "stock", "marketType": "CN", "mc": bson.M{"$gt": 50 * math.Pow(10, 8)},
	}).Sort("-amount").All(&data)
	return data
}

// init klines
func initKline() {
	p := pool.NewPool(8)
	for _, i := range items {
		p.NewTask(func() {
			klineMap.Store(i.Id, i.getKDJKline())
		})
	}
	p.Wait()
}

func (s *Stockx) getKDJKline() []model.Kline {
	var data []model.Kline
	db.KlineDB.Collection(util.CodeToInt(s.Id)).Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"_id.code": s.Id}).
		Sort(bson.M{"_id.time": 1}).
		Do()).All(&data)
	return data
}
