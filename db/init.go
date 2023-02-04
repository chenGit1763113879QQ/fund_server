package db

import (
	"context"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	redisHost = "192.168.0.102:6380"
	mongoHost = "192.168.0.102:27017"
)

var (
	ctx = context.Background()

	LimitDB *redis.Client

	FundDB   *qmgo.Database
	KlineDB  *qmgo.Database
	BackDB   *qmgo.Database
	MinuteDB *qmgo.Database

	Stock  *qmgo.Collection
	Minute *qmgo.Collection
)

// cache
var (
	Numbers    sync.Map
	MainFlow   any
	NorthMoney any
	MarketHot  []bson.M
)

// init database
func init() {
	client, err := qmgo.NewClient(ctx, &qmgo.Config{Uri: "mongodb://" + mongoHost})
	if err != nil {
		panic(err)
	}

	FundDB = client.Database("fund")
	BackDB = client.Database("back")
	KlineDB = client.Database("kline")
	MinuteDB = client.Database("minute")

	Stock = FundDB.Collection("stock")
	Stock.EnsureIndexes(ctx, []string{"symbol"}, []string{"marketType", "type"})

	Minute = FundDB.Collection("minute")
	Minute.EnsureIndexes(ctx, []string{"code,trade_date"}, nil)

	// Redis
	LimitDB = redis.NewClient(&redis.Options{
		Addr: redisHost, DB: 0,
	})
}
