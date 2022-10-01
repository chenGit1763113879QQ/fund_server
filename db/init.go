package db

import (
	"context"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/qiniu/qmgo"
	qmgoOptions "github.com/qiniu/qmgo/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	redisHost = "localhost:6380"
	mongoHost = "localhost:27017"
)

var (
	ctx = context.Background()

	LimitDB *redis.Client

	FundDB   *qmgo.Database
	KlineDB  *qmgo.Database
	BackDB   *qmgo.Database
	MinuteDB *qmgo.Database

	Stock   *qmgo.Collection
	Predict *qmgo.Collection
	Fina    *qmgo.Collection
	Minute  *qmgo.Collection

	User *qmgo.Collection
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
	Stock.EnsureIndexes(ctx, nil, []string{"marketType", "type"})

	Predict = FundDB.Collection("predict")

	Fina = FundDB.Collection("fina")
	Fina.EnsureIndexes(ctx, nil, []string{"ts_code", "end_type"})

	User = FundDB.Collection("user")
	User.EnsureIndexes(ctx, []string{"email", "name"}, nil)

	Minute = FundDB.Collection("minute")
	Minute.EnsureIndexes(ctx, []string{"code,trade_date"}, nil)

	// Redis
	LimitDB = redis.NewClient(&redis.Options{
		Addr: redisHost, DB: 1,
	})
}

func TimeSeriesCollection(name string) *qmgo.Collection {
	// timeSeries option
	tsOpt := new(options.TimeSeriesOptions)
	tsOpt.SetTimeField("time").
		SetGranularity("hours").
		SetMetaField("meta")

	// create collection option
	collOpt := qmgoOptions.CreateCollectionOptions{
		CreateCollectionOptions: options.CreateCollection(),
	}
	collOpt.SetTimeSeriesOptions(tsOpt)

	// create
	KlineDB.CreateCollection(ctx, name, collOpt)

	return KlineDB.Collection(name)
}
