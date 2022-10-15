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
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
)

const pageSize = 20
const XQHOST = "https://stock.xueqiu.com/v5/stock"

var (
	ctx     = context.Background()
	listOpt = bson.M{"members": 0, "pinyin": 0, "lazy_pinyin": 0, "symbol": 0}
)

// GetStockDetail 获取股票详情
func GetStockDetail(code string) bson.M {
	var data bson.M
	db.Stock.Find(ctx, bson.M{"_id": code}).Select(listOpt).One(&data)

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

// GetStockList 获取股票列表
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

// Search 搜索股票内容
func Search(c *gin.Context) {
	input := c.Query("input") + ".*"
	var data []bson.M

	db.Stock.Find(ctx, bson.M{
		"$or": bson.A{
			// regex pattern
			bson.M{"_id": bson.M{"$regex": input, "$options": "i"}},
			bson.M{"name": bson.M{"$regex": input, "$options": "i"}},
			bson.M{"lazy_pinyin": bson.M{"$regex": input, "$options": "i"}},
			bson.M{"pinyin": bson.M{"$regex": input, "$options": "i"}},
		},
	}).Select(listOpt).Sort("marketType", "-type", "-amount").Limit(10).All(&data)

	midware.Success(c, data)
}

// AllBKDetails 所有板块详情
func AllBKDetails(c *gin.Context) {
	var req struct {
		Market util.Code `form:"market" binding:"required"`
		Sort   string    `form:"sort" binding:"required"`
	}
	if err := c.ShouldBind(&req); err != nil {
		midware.Error(c, err)
		return
	}

	data := make([]bson.M, 0)

	db.Stock.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"marketType": req.Market, "type": util.TYPE_IDS}).
		Sort(bson.M{req.Sort: -1}).Limit(50).
		Lookup("stock", "members", "_id", "children").
		Project(bson.M{
			"_id": 1, "name": 1, "pct_chg": 1, "amount": 1, "mc": 1, "followers": 1,
			"children": bson.M{
				"_id": 1, "name": 1, "amount": 1, "pct_chg": 1,
				"price": 1, "mc": 1, "followers": 1,
			},
		}).Do()).All(&data)

	midware.Success(c, data)
}

// PredictKline 获取预测K线结果
func PredictKline(c *gin.Context) {
	data := make([]bson.M, 0)
	period, _ := strconv.Atoi(c.Query("period"))

	db.Predict.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"src_code": c.Query("code"), "period": period}).
		Sort(bson.M{"std": 1}).
		Lookup("stock", "src_code", "_id", "src_code").
		Lookup("stock", "match_code", "_id", "match_code").
		Project(bson.M{
			"_id": 0, "period": 1, "std": 1,
			"start_date": 1, "end_date": 1,
			"src_code": 1, "match_code": 1,
		}).
		Unwind("$src_code").
		Unwind("$match_code").Do()).All(&data)

	midware.Success(c, data)
}

// GetPortfolio 获取雪球自选股
func GetPortfolio(c *gin.Context) {
	url := XQHOST + "/portfolio/stock/list.json?category=1&size=500"
	body, _ := util.XueQiuAPI(url)

	var data []struct {
		Symbol  string `json:"symbol"`
		Created int    `json:"created"`
	}
	util.UnmarshalJSON(body, &data, "data", "stocks")

	symbols := make([]string, len(data))
	for i, s := range data {
		symbols[i] = s.Symbol
	}

	var stocks []bson.M
	db.Stock.Find(ctx, bson.M{"symbol": bson.M{"$in": symbols}}).
		Select(listOpt).All(&stocks)

	midware.Success(c, stocks)
}

// GetHotStock 获取雪球热股
func GetHotStock(c *gin.Context) {
	url := XQHOST + "/hot_stock/list.json?size=100&type=10"
	body, _ := util.XueQiuAPI(url)

	var data []bson.M
	util.UnmarshalJSON(body, &data, "data", "items")

	midware.Success(c, data)
}
