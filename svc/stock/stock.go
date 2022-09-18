package stock

import (
	"context"
	"errors"
	"fund/db"
	"fund/midware"
	"fund/svc/job"
	"fund/util"
	"fund/util/mongox"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	pageSize = 20
)

var (
	ctx     = context.Background()
	listOpt = bson.M{"pinyin": 0, "lazy_pinyin": 0, "members": 0}
)

func GetStockDetail(code string) bson.M {
	var data bson.M
	db.Stock.Find(ctx, bson.M{"_id": code}).Select(bson.M{"members": 0, "pinyin": 0}).One(&data)

	if data != nil {
		var bk []bson.M
		db.Stock.Find(ctx, bson.M{"_id": bson.M{"$in": data["bk"]}}).
			Select(bson.M{"name": 1, "type": 1, "pct_chg": 1}).All(&bk)
		data["bk"] = bk

		for _, i := range job.Markets {
			if i.Market == data["marketType"] {
				data["status"] = i.Status
				data["status_name"] = i.StatusName
				data["trade_date"] = i.TradeTime
				break
			}
		}
	}
	return data
}

func GetStockList(c *gin.Context) {
	var req struct {
		Parent     string   `form:"parent"`
		MarketType uint8    `form:"marketType"`
		Sort       string   `form:"sort"`
		Chart      string   `form:"chart"`
		Page       int64    `form:"page"`
		List       []string `form:"list" json:"list" bson:"list"`
	}
	c.ShouldBind(&req)

	var query qmgo.QueryI

	if req.List != nil {
		query = db.Stock.Find(ctx, bson.M{"_id": bson.M{"$in": req.List}})

	} else if req.Parent != "" {
		var member bson.A
		db.Stock.Find(ctx, bson.M{"_id": req.Parent}).Distinct("members", &member)
		query = db.Stock.Find(ctx, bson.M{"_id": bson.M{"$in": member}})

	} else if req.MarketType > 0 {
		query = db.Stock.Find(ctx, bson.M{
			"marketType": req.MarketType, "type": util.TYPE_STOCK,
		})

	} else {
		midware.Error(c, errors.New("bad request"), http.StatusBadRequest)
		return
	}

	data := make([]bson.M, 0)
	if req.Sort != "" {
		query.Sort(req.Sort)
	}
	if req.Page > 0 {
		query.Skip(pageSize * (req.Page - 1))
	}

	query.Limit(pageSize).Select(listOpt).All(&data)

	// resort
	if req.List != nil {
		for i := range req.List {
			for j := range data {
				if req.List[i] == data[j]["_id"] {
					data[i], data[j] = data[j], data[i]
					break
				}
			}
		}
	}
	midware.Success(c, data)
}

func Search(c *gin.Context) {
	input := c.Query("input") + ".*"
	data := make([]bson.M, 0)

	db.Stock.Find(ctx, bson.M{
		"$or": bson.A{
			// regex pattern
			bson.M{"_id": bson.M{"$regex": input, "$options": "i"}},
			bson.M{"name": bson.M{"$regex": input, "$options": "i"}},
			// allow pinyin
			bson.M{"lazy_pinyin": bson.M{"$regex": input, "$options": "i"}},
			bson.M{"pinyin": bson.M{"$regex": input, "$options": "i"}},
		},
	}).Select(listOpt).Sort("marketType", "-type", "-amount").Limit(10).All(&data)

	// articles
	arts := make([]bson.M, 0)
	db.Article.Find(ctx, bson.M{"title": bson.M{"$regex": input, "$options": "i"}}).
		Sort("-createAt").Limit(8).All(&arts)

	midware.Success(c, bson.M{"stock": data, "arts": arts})
}

func AllBKDetails(c *gin.Context) {
	var req struct {
		Market uint8  `form:"market" binding:"required"`
		Sort   string `form:"sort" binding:"required"`
	}
	if err := c.ShouldBind(&req); err != nil {
		midware.Error(c, err)
		return
	}

	var data []bson.M
	db.Stock.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"marketType": req.Market, "type": util.TYPE_IDS}).
		Lookup("stock", "members", "_id", "children").
		Project(bson.M{
			"name": 1, "pct_chg": 1, "amount": 1, "mc": 1, "followers": 1,
			"children": bson.M{
				"_id": 1, "name": 1, "price": 1, "amount": 1, "pct_chg": 1,
				"mc": 1, "followers": 1,
			},
		}).Sort(bson.M{req.Sort: -1}).Limit(50).Do()).All(&data)

	midware.Success(c, data)
}

func PredictKline(c *gin.Context) {
	data := make([]bson.M, 0)
	db.Predict.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"预测股票": c.Query("code")}).
		Sort(bson.M{"标准差": -1}).
		Limit(10).
		Lookup("stock", "预测股票", "_id", "预测股票").
		Lookup("stock", "匹配股票", "_id", "匹配股票").
		Project(bson.M{
			"_id": 0, "匹配天数": 1, "标准差": 1, "匹配日期": 1,
			"预测股票": bson.M{"_id": 1, "name": 1}, "匹配股票": bson.M{"_id": 1, "name": 1},
		}).
		Unwind("$预测股票").
		Unwind("$匹配股票").Do()).All(&data)
	midware.Success(c, data)
}
