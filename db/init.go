package db

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/qiniu/qmgo"
)

// 生产环境
const (
// redisHost    = "redis:6379"
// mongoHost    = "mongo:27017"
)

// 本地环境
const (
	redisHost = "localhost:6380"
	mongoHost = "localhost:27017"
)

var (
	ctx = context.Background()

	LimitDB *redis.Client // 流量控制

	FundDB   *qmgo.Database
	KlineDB  *qmgo.Database
	MinuteDB *qmgo.Database

	Stock    *qmgo.Collection // 行情
	Predict  *qmgo.Collection // 预测
	Backtest *qmgo.Collection // 回测
	Events   *qmgo.Collection // 重大事项
	Fina     *qmgo.Collection // 财务指标

	User    *qmgo.Collection // 用户
	Article *qmgo.Collection // 文章
)

// init database
func init() {
	client, err := qmgo.NewClient(ctx, &qmgo.Config{Uri: "mongodb://" + mongoHost})
	if err != nil {
		panic(err)
	}

	FundDB = client.Database("fund")
	KlineDB = client.Database("kline")
	MinuteDB = client.Database("minute")

	Stock = FundDB.Collection("stock")
	Stock.EnsureIndexes(ctx, nil, []string{"marketType", "type"})

	Predict = FundDB.Collection("predict")
	Backtest = FundDB.Collection("backtest")

	Events = FundDB.Collection("events")
	Events.EnsureIndexes(ctx, nil, []string{"type", "ts_code", "ann_date"})

	Fina = FundDB.Collection("fina")
	Fina.EnsureIndexes(ctx, nil, []string{"ts_code", "end_type"})

	User = FundDB.Collection("user")
	User.EnsureIndexes(ctx, []string{"email", "name"}, nil)

	Article = FundDB.Collection("article")
	Article.EnsureIndexes(ctx, []string{"uid,title"}, []string{"createAt"})

	// Redis
	LimitDB = redis.NewClient(&redis.Options{
		Addr: redisHost, DB: 1,
	})
}
