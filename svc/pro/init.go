package pro

import (
	"context"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/pool"
	"math"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Stockx struct {
	Id     string `bson:"_id"`
	Market string `bson:"marketType"`
	Name   string
	Mc     float64
}

var (
	ctx      = context.Background()
	KlineMap = sync.Map{}
)

func initStock() []Stockx {
	var data []Stockx
	db.Stock.Find(ctx, bson.M{
		"type": "stock", "marketType": "CN", "mc": bson.M{"$gt": 50 * math.Pow(10, 8)},
	}).Sort("-amount").All(&data)
	return data
}

func initKline() {
	p := pool.NewPool(8)
	items := initStock()

	for _, i := range items {
		p.NewTask(func() {
			KlineMap.Store(i.Id, i.getKDJKline())
		})
	}
	p.Wait()
}

func (s *Stockx) getKDJKline() []model.Kline {
	var data []model.Kline
	db.KlineDB.Collection(util.CodeToInt(s.Id)).Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"_id.code": s.Id}}},
		bson.D{{Key: "$sort", Value: bson.M{"_id.time": 1}}},
	}).All(&data)
	return data
}
