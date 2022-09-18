package stock

import (
	"errors"
	"fund/db"
	"fund/midware"
	"fund/model"
	"fund/svc/job"
	"fund/util"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

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

		i := model.Stock{}

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

	// 市场总览
	jobMarket := func() {
		temp, _ := db.Numbers.Load(req.MarketType)
		ws.WriteJson(bson.M{"market": bson.M{
			"0": temp, "1": db.NorthMoney, "2": db.MainFlow, "3": db.MarketHot,
		}})
	}

	// 板块行情
	jobBK := func() {
		dt := make([]bson.M, 0)
		listOpt := bson.M{
			"name": 1, "pct_chg": 1, "main_net": 1, "pct_leader": 1, "main_net_leader": 1,
		}

		switch req.BkType {
		case "I":
			db.Stock.Find(ctx, bson.M{
				"marketType": req.MarketType,
				"type":       util.TYPE_IDS,
			}).Sort(req.BkSort).Select(listOpt).Limit(6).All(&dt)

		case "Map":
			db.Stock.Find(ctx, bson.M{
				"marketType": req.MarketType,
				"type":       util.TYPE_IDS,
			}).Sort("-amount").Select(listOpt).Limit(15).All(&dt)
		}
		ws.WriteJson(bson.M{"bk": dt})
	}

	// 监听
	go func() {
		for ws.Err == nil {
			ws.ReadJson(&req)
			jobBK()
			jobMarket()
		}
	}()

	job.Cond.L.Lock()

	for ws.Err == nil {
		jobMarket()
		jobBK()

		time.Sleep(time.Second)
		job.Cond.Wait()
	}
	job.Cond.L.Unlock()
}

func Notify(c *gin.Context) {
	ws := model.NewWebSocket(c)

	var req struct {
		Code  string `json:"code"`
		Chart string `json:"chart"`
	}

	for ws.Err == nil {
		ws.ReadJson(&req)

		switch req.Chart {
		default:
			ws.WriteJson(bson.M{"data": GetSimpleChart(req.Code, req.Chart)})
		}
	}
}
