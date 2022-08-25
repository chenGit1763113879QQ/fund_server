package stock

import (
	"context"
	"errors"
	"fmt"
	"fund/cache"
	"fund/db"
	"fund/midware"
	"fund/model"
	"fund/svc/job"
	"fund/util"
	"fund/util/mongox"
	"fund/util/pool"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocarina/gocsv"
	jsoniter "github.com/json-iterator/go"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	pr "go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	pageSize = 20
	EMHOST   = "http://push2.eastmoney.com/api/qt/stock" // eastmoney host
)

var (
	json    = jsoniter.ConfigCompatibleWithStandardLibrary
	ctx     = context.Background()
	listOpt = bson.M{
		"_id": 1, "cid": 1, "type": 1, "marketType": 1, "price": 1, "pct_chg": 1, "vol": 1, "amount": 1,
		"main_net": 1, "name": 1, "net": 1, "basic_eps": 1, "roe": 1, "tr": 1, "pct_rate": 1,
		"mc": 1, "netprofit_yoy": 1, "or_yoy": 1, "ratio": 1, "pe_ttm": 1, "pct_year": 1,
	}
)

// get stock detail
func GetStock(code string) bson.M {
	var data bson.M
	db.Stock.Find(ctx, bson.M{"_id": code}).Select(bson.M{"members": 0, "company": 0}).One(&data)
	// 添加详情
	if data != nil {
		// 板块
		var bk []bson.M
		db.Stock.Find(ctx, bson.M{"_id": bson.M{"$in": data["bk"]}}).
			Select(bson.M{"name": 1, "type": 1, "pct_chg": 1}).All(&bk)
		data["bk"] = bk

		// 市场状态
		for _, i := range job.Markets {
			if i.Name == data["marketType"] {
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

// get stock list
func GetStockList(c *gin.Context) {
	req := new(model.CListOpt)
	c.ShouldBind(req)

	var query qmgo.QueryI

	if req.List != nil {
		query = db.Stock.Find(ctx, bson.M{"_id": bson.M{"$in": req.List}})

	} else if req.Parent != "" {
		var member bson.A
		db.Stock.Find(ctx, bson.M{"_id": req.Parent}).Distinct("members", &member)
		query = db.Stock.Find(ctx, bson.M{"_id": bson.M{"$in": member}})

	} else if req.MarketType != "" {
		query = db.Stock.Find(ctx, bson.M{
			"marketType": req.MarketType, "type": "stock", "price": bson.M{"$gt": 0},
		})

	} else {
		midware.Error(c, errors.New("bad request"), http.StatusBadRequest)
		return
	}

	data := make([]bson.M, 0)
	if req.Sort != "" {
		query = query.Sort(req.Sort)
	}
	if req.Page > 0 {
		query = query.Skip(pageSize * (req.Page - 1))
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

// get active list
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

// get minute kline
func GetMinute(code string) any {
	var data []struct {
		Price float64 `json:"price"`
		Avg   float64 `json:"avg"`
		Net   float64 `json:"net"`
		Huge  float64 `json:"huge"`
		Big   float64 `json:"big"`
		Mid   float64 `json:"mid"`
		Small float64 `json:"small"`
		Time  int64   `json:"time"`
		Vol   int64   `json:"vol"`
		Id    struct {
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

// add simple chart data
func AddChart(chart string, items []bson.M) {
	var f func(item bson.M, arg string)

	switch chart {
	case "60day":
		// 60day trends
		f = func(item bson.M, arg string) {
			code := item["_id"].(string)

			go job.GetKline(item)

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

	case "price", "net", "main_net":
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

// search data
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

// get ticks details
func GetRealTicks(item bson.M) bson.M {
	cid, _ := item["cid"].(string)

	var pankou bson.M
	var ticks []*struct {
		Time  string  `csv:"time" json:"time"`
		Price float64 `csv:"price" json:"price"`
		Vol   int     `csv:"vol" json:"vol"`
		Type  int     `csv:"type" json:"type"`
	}
	p := pool.NewPool(2)

	// pankou
	p.NewTask(func() {
		if item["marketType"] == "CN" {
			body, _ := util.GetAndRead(EMHOST + "/get?fltt=2&fields=f530&secid=" + cid)
			json.Get(body, "data").ToVal(&pankou)
		}
	})

	// ticks
	p.NewTask(func() {
		body, _ := util.GetAndRead(EMHOST + "/details/get?fields1=f1&fields2=f51,f52,f53,f55&pos=-30&secid=" + cid)

		var info []string
		json.Get(body, "data", "details").ToVal(&info)

		gocsv.Unmarshal(strings.NewReader("time,price,vol,type\n"+strings.Join(info, "\n")), &ticks)
		for _, i := range ticks {
			switch i.Type {
			case 4:
				i.Type = 0
			case 1:
				i.Type = -1
			case 2:
				i.Type = 1
			}
		}
	})
	p.Wait()

	return bson.M{"ticks": ticks, "pankou": pankou}
}

// get kline data
func GetKline(c *gin.Context) {
	req := new(model.KlineOpt)
	c.ShouldBind(req)

	var items, groupById bson.M

	db.Stock.Find(ctx, bson.M{"_id": req.Code}).One(&items)
	if items == nil {
		midware.Error(c, errors.New("code not found"))
		return
	}

	// 数据是否需要更新
	go job.GetKline(items)

	// 分钟行情
	if strings.Contains(req.Period, "min") {
		val := strings.Split(req.Period, "min")[0]
		duration, err := strconv.Atoi(val)
		if err != nil {
			midware.Error(c, err)
			return
		}

		groupById = bson.M{"$subtract": bson.A{
			bson.M{"$subtract": bson.A{"$time", time.Time{}}},
			bson.M{"$mod": bson.A{
				bson.M{"$subtract": bson.A{"$time", time.Time{}}},
				duration * 1000 * 60,
			}},
		}}
	} else {
		format := map[string]string{
			"d": "%Y-%m-%d", "w": "%Y-%V", "m": "%Y-%m", "y": "%Y",
		}[req.Period]

		groupById = bson.M{"$dateToString": bson.M{"format": format, "date": "$time"}}
	}

	if req.StartDate == "" {
		switch req.Period {
		case "y", "q", "m":
			req.StartDate = "2000-01-01"
		case "w":
			req.StartDate = "2016-06-01"
		default:
			req.StartDate = "2019-06-01"
		}
	}
	t, _ := time.Parse("2006-01-02", req.StartDate)

	// 分组聚合
	group := bson.M{
		"_id":         groupById,
		"time":        bson.M{"$last": "$time"},
		"open":        bson.M{"$first": "$open"},
		"close":       bson.M{"$last": "$close"},
		"high":        bson.M{"$max": "$high"},
		"low":         bson.M{"$min": "$low"},
		"ratio":       bson.M{"$last": "$ratio"},
		"main_net":    bson.M{"$sum": "$main_net"},
		"vol":         bson.M{"$sum": "$vol"},
		"amount":      bson.M{"$sum": "$amount"},
		"pct_chg":     bson.M{"$sum": "$pct_chg"},
		"tr":          bson.M{"$sum": "$tr"},
		"rzrqye":      bson.M{"$last": "$rzrqye"},
		"winner_rate": bson.M{"$last": "$winner_rate"},
	}

	// query
	var data []bson.M

	fmt.Println(req.Code, util.Md5Code(req.Code))

	db.KlineDB.Collection(util.Md5Code(req.Code)).Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"code": req.Code, "time": bson.M{"$gt": t}}).
		Group(group).Sort(bson.M{"time": 1}).Do()).All(&data)

	if req.Head > 0 {
		midware.Success(c, data[:req.Head])
		return

	} else if req.Tail > 0 {
		midware.Success(c, data[len(data)-req.Tail:])
		return
	}
	midware.Success(c, data)
}

// data center
// includes topList, blockTrade, events
func DataCenter(c *gin.Context) {
	var req struct {
		Code      string `form:"code"`
		TradeDate string `form:"trade_date"`
		Symbol    string `form:"symbol"`
		Type      string `form:"type"`
	}
	c.ShouldBind(&req)

	// 解析参数
	filter := bson.M{}
	if req.Code != "" {
		filter["ts_code"] = req.Code
	}
	if req.TradeDate != "" {
		t, _ := time.Parse(time.RFC3339, req.TradeDate)
		filter["trade_date"] = t.Format("20060102")
	}
	if req.Type != "" {
		filter["type"] = req.Type
	}

	data := make([]bson.M, 0)

	switch c.Param("path") {
	case "topList":
		db.FundDB.Collection("topList").Find(ctx, filter).Sort("-trade_date").
			Select(bson.M{"_id": 0}).Limit(150).All(&data)

	case "blockTrade":
		db.FundDB.Collection("blockTrade").Aggregate(ctx, mongox.Pipeline().
			Match(filter).
			Lookup("stock", "ts_code", "_id", "stock").
			Unwind("$stock").
			Project(bson.M{
				"_id": 0, "name": "$stock.name", "trade_date": 1,
				"buyer": 1, "seller": 1, "ts_code": 1, "price": 1, "vol": 1, "amount": 1,
			}).
			Sort(bson.M{"trade_date": -1}).
			Limit(50).Do()).All(&data)

	case "events":
		db.FundDB.Collection("events").Aggregate(ctx, mongox.Pipeline().
			Match(filter).
			Lookup("stock", "ts_code", "_id", "stock").
			Unwind("$stock").
			Project(bson.M{
				"_id": 0, "name": "$stock.name", "ann_date": 1, "end_date": 1,
				"vol": 1, "amount": 1, "ts_code": 1, "high_limit": 1,
			}).
			Sort(bson.M{"ann_date": -1}).
			Limit(50).Do()).All(&data)

	case "fina":
		db.Fina.Find(ctx, bson.M{"ts_code": req.Code, "end_type": "4"}).
			Sort("end_date").Select(bson.M{"_id": 0}).All(&data)

	default:
		midware.Error(c, errors.New("page not found"), http.StatusNotFound)
		return
	}

	midware.Success(c, data)
}

// get all stock
func GetAllStock(c *gin.Context) {
	data := make([]bson.M, 0)
	params := make(map[string]string)
	c.ShouldBind(params)

	err := db.Stock.Find(ctx, params).Select(bson.M{"name": 1, "type": 1, "marketType": 1}).
		Sort("marketType", "-type", "-amount").All(&data)
	midware.Auto(c, err, data)
}

// get market bk details
func DetailBK(c *gin.Context) {
	var data []bson.M
	db.Stock.Find(ctx, bson.M{"type": c.Query("type")}).Select(bson.M{
		"name": 1, "pct_chg": 1, "pct_leader": 1, "main_net_leader": 1, "main_net": 1, "net": 1, "count": 1,
	}).All(&data)
	midware.Success(c, data)
}

// get predict klines data
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

// get groups data
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

// add group
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

// remove group
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

// change stock groups
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

// stock in groups
func InGroup(c *gin.Context) {
	code := c.Query("code")
	if cache.Stock.Exist(code) {
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

// update active list
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
