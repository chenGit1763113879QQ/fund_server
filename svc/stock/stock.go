package stock

import (
	"context"
	"errors"
	"fund/db"
	"fund/midware"
	"fund/model"
	"fund/svc/job"
	"fund/util"
	"fund/util/mongox"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	pr "go.mongodb.org/mongo-driver/bson/primitive"
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

func getStockList(codeStr string, chart string) []bson.M {
	codes := strings.Split(codeStr, ",")

	data := make([]bson.M, 0)
	db.Stock.Find(ctx, bson.M{"_id": bson.M{"$in": codes}}).Select(listOpt).All(&data)

	// resort
	for i := range codes {
		for j := range data {
			if codes[i] == data[j]["_id"] {
				data[i], data[j] = data[j], data[i]
				break
			}
		}
	}
	AddChart(chart, data)
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
	AddChart(req.Chart, data)
	midware.Success(c, data)
}

func GetActiveList(c *gin.Context) {
	req := new(model.Group)
	chart := c.Query("chart")
	c.ShouldBind(req)

	filter := bson.M{"marketType": req.Market, "type": "stock"}

	// parents
	if len(req.Parent) > 0 {
		members := make([]string, 0)
		db.Stock.Find(ctx, bson.M{"_id": bson.M{"$in": req.Parent}}).Distinct("members", &members)
		filter["_id"] = bson.M{"$in": members}
	}

	// select range
	ranges := append(req.MarketRange, req.FinaRange...)
	for _, r := range ranges {
		if r.Left != 0 || r.Right != 0 {
			ind := bson.M{}
			// left range
			if r.Left != 0 {
				ind["$gte"] = util.Exp(r.Unit > 0, r.Left*r.Unit, r.Left)
			}
			// right range
			if r.Right != 0 {
				ind["$lte"] = util.Exp(r.Unit > 0, r.Right*r.Unit, r.Right)
			}
			filter[r.Name] = ind
		}
	}

	data := make([]bson.M, 0)
	db.Stock.Find(ctx, filter).Select(listOpt).Sort("-amount").Limit(10).All(&data)

	AddChart(chart, data)
	midware.Success(c, data)
}

func GetMinute(code string) any {
	var data []struct {
		Price   float64 `json:"price"`
		Avg     float64 `json:"avg"`
		MainNet float64 `json:"main_net"`
		Time    int64   `json:"time"`
		Vol     int64   `json:"vol"`
		Id      struct {
			Time int64
		} `json:"-" bson:"_id"`
	}

	db.MinuteDB.Collection(job.GetTradeTime(code).Format("2006/01/02")).
		Find(ctx, bson.M{"_id.code": code}).All(&data)
	for i := range data {
		data[i].Time = data[i].Id.Time
	}
	return data
}

func AddChart(chart string, items []bson.M) {
	var f func(item bson.M, arg string)

	switch chart {
	case "60day":
		// 60day trends
		f = func(item bson.M, arg string) {
			code := item["_id"].(string)

			var data []struct {
				Close float64 `bson:"close"`
			}

			db.KlineDB.Collection(util.Md5Code(code)).Find(ctx, bson.M{"_id.code": code}).
				Sort("-_id.time").Select(bson.M{"close": 1}).Limit(60).All(&data)

			price := make([]float64, len(data))
			for i := 0; i < len(data); i++ {
				price[len(data)-i-1] = data[i].Close
			}

			item["chart"] = bson.M{"total": 60, "value": price, "type": "line"}
		}

	case "price", "main_net":
		// simple chart
		f = func(item bson.M, arg string) {
			var data []struct {
				Price   float64 `bson:"price"`
				Net     float64 `bson:"net"`
				MainNet float64 `bson:"main_net"`
			}

			var total, sp = 0, 6
			switch item["marketType"] {
			case "US":
				total = 390 / sp
			case "HK":
				total = 310 / sp
			case "CN":
				sp = 5
				total = 240 / sp
			}

			// query
			db.MinuteDB.Collection(job.GetTradeTime(item["_id"].(string)).Format("2006/01/02")).
				Find(ctx, bson.M{"_id.code": item["_id"], "minutes": bson.M{"$mod": bson.A{sp, 0}}}).
				Select(bson.M{arg: 1}).All(&data)

			// array
			arr := make([]float64, len(data)+1)
			for i := range data {
				arr[i] = data[i].Price
			}
			arr[len(arr)-1], _ = item["price"].(float64)

			item["chart"] = bson.M{"total": total + 1, "value": arr, "type": "line"}
		}
	default:
		return
	}
	for i := range items {
		f(items[i], chart)
	}
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

func GetKline(c *gin.Context) {
	var req struct {
		Code      string `form:"code" binding:"required"`
		Period    string `form:"period" binding:"required"`
		StartDate string `form:"start_date"`
		Head      int    `form:"head"`
		Tail      int    `form:"tail"`
	}
	if err := c.ShouldBind(&req); err != nil {
		midware.Error(c, err)
		return
	}

	var items bson.M

	db.Stock.Find(ctx, bson.M{"_id": req.Code}).One(&items)
	if items == nil {
		midware.Error(c, errors.New("code not found"))
		return
	}

	format := map[string]string{
		"d": "%Y/%m/%d", "w": "%Y/%V", "m": "%Y/%m", "y": "%Y",
	}[req.Period]

	if req.StartDate == "" {
		switch req.Period {
		case "y", "q", "m":
			req.StartDate = "2000-01-01"
		case "w":
			req.StartDate = "2013-01-01"
		default:
			req.StartDate = "2017-09-01"
		}
	}

	t, _ := time.Parse("2006-01-02", req.StartDate)

	var data []bson.M
	db.KlineDB.Collection(util.Md5Code(req.Code)).Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"code": req.Code, "time": bson.M{"$gt": t}}).
		Group(bson.M{
			"_id":      bson.M{"$dateToString": bson.M{"format": format, "date": "$time"}},
			"time":     bson.M{"$last": "$time"},
			"open":     bson.M{"$first": "$open_qfq"},
			"close":    bson.M{"$last": "$close_qfq"},
			"high":     bson.M{"$max": "$high_qfq"},
			"low":      bson.M{"$min": "$low_qfq"},
			"ratio":    bson.M{"$last": "$ratio"},
			"main_net": bson.M{"$sum": "$main_net"},
			"vol":      bson.M{"$sum": "$vol"},
			"amount":   bson.M{"$sum": "$amount"},
			"pct_chg":  bson.M{"$sum": "$pct_chg"},
			"tr":       bson.M{"$sum": "$tr"},
			// "rzrqye":      bson.M{"$last": "$rzrqye"},
			"winner_rate": bson.M{"$last": "$winner_rate"},
		}).
		Match(bson.M{"close": bson.M{"$gt": 0}}).
		Sort(bson.M{"time": 1}).Do()).All(&data)

	if req.Head > 0 {
		midware.Success(c, data[:req.Head])

	} else if req.Tail > 0 {
		midware.Success(c, data[len(data)-req.Tail:])

	} else {
		midware.Success(c, data)
	}
}

func GetAllStock(c *gin.Context) {
	data := make([]bson.M, 0)
	params := make(map[string]string)
	c.ShouldBind(params)

	err := db.Stock.Find(ctx, params).Select(bson.M{"name": 1, "type": 1, "marketType": 1}).
		Sort("marketType", "-type", "-amount").All(&data)
	midware.Auto(c, err, data)
}

func DetailBK(c *gin.Context) {
	var data []bson.M
	db.Stock.Find(ctx, bson.M{"type": c.Query("type")}).Select(bson.M{
		"name": 1, "pct_chg": 1, "amount": 1, "main_net": 1,
	}).All(&data)
	midware.Success(c, data)
}

func DetailBKGlobal(c *gin.Context) {
	var data []bson.M
	db.Stock.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"marketType": util.MARKET_CN, "type": util.TYPE_I1}).
		Lookup("stock", "members", "_id", "children").
		Project(bson.M{
			"name": 1, "pct_chg": 1, "amount": 1, "mc": 1, "count": 1,
			"children": bson.M{"_id": 1, "name": 1, "amount": 1, "pct_chg": 1, "mc": 1},
		}).Do()).All(&data)
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

func getGroups(id pr.ObjectID) *model.Groups {
	g := new(model.Groups)
	db.User.Find(ctx, bson.M{"_id": id}).One(g)
	return g
}

func GetGroups(c *gin.Context) {
	uid := c.MustGet("id").(pr.ObjectID)
	chart := c.Query("chart")

	var data model.Groups
	db.User.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"_id": uid}).
		Lookup("stock", "groups.list", "_id", "stocks").
		Project(bson.M{"groups": 1, "stocks": listOpt}).Do()).One(&data)

	AddChart(chart, data.Stocks)
	midware.Success(c, data)
}

func AddGroup(c *gin.Context) {
	req := new(model.Group)
	c.ShouldBind(req)
	uid := c.MustGet("id").(pr.ObjectID)

	if req.Name == "" {
		midware.Error(c, errors.New("分组名为空"))
		return
	}

	g := getGroups(uid)
	for _, i := range g.Groups {
		if i.Name == req.Name {
			midware.Error(c, errors.New("分组名重复"))
			return
		}
	}

	if len(g.Groups) >= 6 {
		midware.Error(c, errors.New("最多创建六个分组"))
		return
	}

	if !req.IsActive {
		req.List = []string{}
	}

	err := db.User.UpdateId(ctx, uid, bson.M{"$addToSet": bson.M{"groups": req}})
	midware.Auto(c, err, "新建分组成功")
}

func RemGroup(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBind(&req); err != nil {
		midware.Error(c, err)
		return
	}

	uid := c.MustGet("id").(pr.ObjectID)
	g := getGroups(uid)

	for _, gr := range g.Groups {
		if gr.Name == req.Name && gr.IsSys {
			midware.Error(c, errors.New("不能删除该分组"))
			return
		}
	}

	err := db.User.UpdateId(ctx, uid, bson.M{
		"$pull": bson.M{"groups": bson.M{"name": req.Name}},
	})

	midware.Auto(c, err, "已删除分组")
}

func ChangeGroup(c *gin.Context) {
	var req struct {
		Code    string   `json:"code" binding:"required"`
		InGroup []string `json:"inGroup" binding:"required"`
	}
	if err := c.ShouldBind(&req); err != nil {
		midware.Error(c, err)
		return
	}

	uid := c.MustGet("id").(pr.ObjectID)

	group := getGroups(uid)
	bulk := db.User.Bulk()

	for _, g := range group.Groups {
		if g.IsSys || g.IsActive {
			continue
		}
		filter := bson.M{"_id": uid, "groups.name": g.Name}
		// del
		bulk.UpdateOne(filter, bson.M{"$pull": bson.M{"groups.$.list": req.Code}})
		if util.In(g.Name, req.InGroup) {
			// add
			bulk.UpdateOne(filter, bson.M{"$push": bson.M{"groups.$.list": bson.M{"$each": bson.A{req.Code}, "$position": 0}}})
		}
	}

	_, err := bulk.Run(ctx)
	midware.Auto(c, err, "修改分组成功")
}

func InGroup(c *gin.Context) {
	code := c.Query("code")
	res, _ := db.Stock.Find(ctx, bson.M{"_id": code}).Count()
	if res == 0 {
		midware.Error(c, errors.New("code not exist"))
		return
	}

	uid := c.MustGet("id").(pr.ObjectID)
	groups := getGroups(uid)

	// 获取最近浏览长度
	var length int
	for _, g := range groups.Groups {
		if g.Name == "最近浏览" {
			length = len(g.List)
		}
	}
	bulk := db.User.Bulk()
	filter := bson.M{"_id": uid, "groups.name": "最近浏览"}

	if length >= 50 {
		bulk.UpdateOne(filter, bson.M{"$pop": bson.M{"groups.$.list": 1}})
	}

	// 添加至最近浏览
	bulk.UpdateOne(filter, bson.M{"$pull": bson.M{"groups.$.list": code}}).
		UpdateOne(filter, bson.M{"$push": bson.M{"groups.$.list": bson.M{"$each": bson.A{code}, "$position": 0}}}).
		Run(ctx)

	allGroup := make([]string, 0)
	inGroup := make([]string, 0)

	for _, g := range groups.Groups {
		if g.IsSys || g.IsActive {
			continue
		}
		allGroup = append(allGroup, g.Name)
		if util.In(code, g.List) {
			inGroup = append(inGroup, g.Name)
		}
	}

	midware.Success(c, bson.M{"allGroup": allGroup, "inGroup": inGroup})
}

func PutActiveList(c *gin.Context) {
	req := new(model.Group)
	c.ShouldBind(req)
	uid := c.MustGet("id").(pr.ObjectID)

	req.IsActive = true

	err := db.User.UpdateOne(ctx, bson.M{"_id": uid, "groups.name": req.Name}, bson.M{
		"$set": bson.M{"groups.$": req},
	})
	midware.Auto(c, err, nil)
}
