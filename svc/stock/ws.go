package stock

import (
	"errors"
	"fund/cache"
	"fund/db"
	"fund/midware"
	"fund/model"
	"fund/svc/job"
	"fund/util"
	"fund/util/mongox"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	pr "go.mongodb.org/mongo-driver/bson/primitive"
)

// 自选列表
func ConnectCList(c *gin.Context) {
	ws := model.NewWebSocket(c)
	defer ws.Conn.Close()

	type ActiveReq struct {
		Market      string        `json:"market"`
		Parent      []string      `json:"parent"`
		MarketRange []model.Range `json:"market_range"`
		FinaRange   []model.Range `json:"fina_range"`
	}
	var req struct {
		Type   string    `json:"type"`
		Chart  string    `form:"chart" json:"chart"`
		List   []string  `json:"list"`
		Active ActiveReq `json:"active"`
	}
	c.ShouldBind(&req)

	uid := c.MustGet("id").(pr.ObjectID)

	// init
	init := func() {
		var data model.Groups

		db.User.Aggregate(ctx, mongox.Pipeline().
			Match(bson.M{"_id": uid}).
			Lookup("stock", "groups.list", "_id", "stocks").
			Project(bson.M{"groups": 1, "stocks": listOpt}).Do()).
			One(&data)

		AddChart(req.Chart, data.Stocks)
		ws.WriteJson(data)
	}

	// active list
	getActiveList := func(opt ActiveReq) {
		filter := bson.M{"marketType": opt.Market, "type": "stock"}

		// parents
		if opt.Market == "CN" && len(opt.Parent) > 0 {
			con := make([]string, 0)
			db.Stock.Find(ctx, bson.M{"_id": bson.M{"$in": opt.Parent}}).Distinct("members", &con)

			filter["_id"] = bson.M{"$in": con}
		}

		// select range
		ranges := append(opt.MarketRange, opt.FinaRange...)
		for _, r := range ranges {
			if r.Left != 0 || r.Right != 0 {
				ind := bson.M{}
				// 左区间
				if r.Left != 0 {
					ind["$gte"] = util.Exp(r.Unit > 0, r.Left*r.Unit, r.Left)
				}
				// 右区间
				if r.Right != 0 {
					ind["$lte"] = util.Exp(r.Unit > 0, r.Right*r.Unit, r.Right)
				}
				filter[r.Name] = ind
			}
		}

		data := make([]bson.M, 0)
		db.Stock.Find(ctx, filter).Select(listOpt).Sort("-amount").Limit(10).All(&data)

		AddChart(req.Chart, data)
		ws.WriteJson(bson.M{"active": data})
	}

	// listen
	go func() {
		init()
		for ws.Err == nil {
			ws.ReadJson(&req)

			switch req.Type {
			case "refresh":
				init()
			case "active":
				getActiveList(req.Active)
			}
		}
	}()

	priceCache := map[string]float64{}

	// send
	job.Cond.L.Lock()
	for ws.Err == nil {
		job.Cond.Wait()

		switch req.Type {
		case "list":
			data := cache.Stock.Loads(req.List)
			for _, i := range data {
				if i.Price != priceCache[i.Id] {
					priceCache[i.Id] = i.Price

					ws.WriteBson(i)
				}
			}
		case "active":
			getActiveList(req.Active)
			time.Sleep(time.Second * 2)
		}
	}
	job.Cond.L.Unlock()
}

// 股票详情
func ConnectItems(c *gin.Context) {
	ws := model.NewWebSocket(c)
	defer ws.Conn.Close()

	code := c.Query("code")
	items := GetStockDetail(code)
	if items == nil {
		midware.Error(c, errors.New("代码不存在"))
		return
	}

	// 热度
	go db.Stock.UpdateId(ctx, code, bson.M{"$inc": bson.M{"view": 1}})

	ws.WriteJson(bson.M{"items": items})
	go ws.WriteJson(bson.M{"minute": GetMinute(code)})

	// 最新资讯
	go func() {
		var news bson.M
		db.Article.Find(ctx, bson.M{"tag": code, "type": 2}).Sort("-createAt").One(&news)
		ws.WriteJson(bson.M{"news": news})
	}()

	job.Cond.L.Lock()
	// 更新
	for ws.Err == nil {
		job.Cond.Wait()

		i := cache.Stock.Load(code)

		if i.Vol != items["vol"] {
			ws.WriteBson(bson.M{"items": i})
			// update cache
			items["vol"] = i.Vol
		}
	}
	job.Cond.L.Unlock()
}

// 市场行情
func ConnectMarket(c *gin.Context) {
	ws := model.NewWebSocket(c)
	var req struct {
		MarketType string `form:"marketType" json:"marketType"`
		BkSort     string `form:"bkSort" json:"bkSort"`
		BkType     string `form:"bkType" json:"bkType"`
	}
	c.ShouldBind(&req)

	// 市场指数
	jobIndex := func() {
		data := make([]bson.M, 0)
		switch req.MarketType {
		case "CN":
			data = getStockList("000001.SH,399001.SZ,399006.SZ", "price")
		case "HK":
			data = getStockList("HSI.HK,HSCEI.HK,HSTECH.HK", "price")
		case "US":
			data = getStockList("NDX.US,DJIA.US,SPX.US", "price")
		}
		ws.WriteJson(bson.M{"index": data})
	}

	// 市场总览
	jobMarket := func() {
		temp, _ := cache.Numbers.Load(req.MarketType)
		ws.WriteJson(bson.M{"market": bson.M{
			"0": temp, "1": cache.NorthMoney, "2": cache.MainFlow, "3": cache.MarketHot,
		}})
	}

	// 板块行情
	jobBK := func() {
		if req.MarketType != "CN" {
			return
		}
		dt := make([]bson.M, 0)
		listOpt := bson.M{
			"name": 1, "pct_chg": 1, "main_net": 1, "pct_leader": 1, "main_net_leader": 1,
		}

		switch req.BkType {
		case "I":
			db.Stock.Find(ctx, bson.M{"type": bson.M{"$in": bson.A{"I1", "I2"}}}).
				Sort(req.BkSort).Select(listOpt).Limit(6).All(&dt)

		case "C":
			db.Stock.Find(ctx, bson.M{"type": "C"}).
				Sort(req.BkSort).Select(listOpt).Limit(6).All(&dt)

		case "Map":
			db.Stock.Find(ctx, bson.M{"type": bson.M{"$in": bson.A{"I1", "I2"}}}).
				Sort("-amount").Select(listOpt).Limit(15).All(&dt)
		}
		ws.WriteJson(bson.M{"bk": dt})
	}

	// 监听
	go func() {
		for ws.Err == nil {
			ws.ReadJson(&req)

			jobIndex()
			jobBK()
			jobMarket()
		}
	}()

	job.Cond.L.Lock()

	for ws.Err == nil {
		jobIndex()
		jobMarket()
		jobBK()

		time.Sleep(time.Second)
		job.Cond.Wait()
	}
	job.Cond.L.Unlock()
}
