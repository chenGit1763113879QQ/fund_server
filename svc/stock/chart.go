package stock

import (
	"errors"
	"fund/db"
	"fund/midware"
	"fund/svc/job"
	"fund/util"
	"fund/util/mongox"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func GetMinute(code string) any {
	var data []struct {
		Price   float64 `json:"price"`
		PctChg  float64 `json:"pct_chg" bson:"pct_chg"`
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

func GetSimpleChart(code string, chartType string) any {
	var data struct {
		Total uint    `json:"total"`
		Value any     `json:"value"`
		Open  float64 `json:"open"`
		Close float64 `json:"close"`
	}

	switch chartType {
	case "minute":
		var arr []struct {
			Time   int64   `json:"time"`
			PctChg float64 `json:"value" bson:"pct_chg"`
			Id     struct {
				Time int64
			} `json:"-" bson:"_id"`
		}

		db.MinuteDB.Collection(job.GetTradeTime(code).Format("2006/01/02")).
			Find(ctx, bson.M{"_id.code": code}).All(&arr)

		for i := range arr {
			arr[i].Time = arr[i].Id.Time
		}

		data.Total = 240
		data.Value = arr
		data.Open = 0
		if len(arr) > 0 {
			data.Close = arr[len(arr)-1].PctChg
		}

	case "60day":
		var arr []struct {
			Time  time.Time `json:"time"`
			Close float64   `json:"value" bson:"close"`
		}

		t, _ := time.Parse("2006/01/02", "2022/01/01")

		db.KlineDB.Collection(util.Md5Code(code)).
			Find(ctx, bson.M{"code": code, "time": bson.M{"$gt": t}}).
			Sort("-time").Select(bson.M{"close": 1, "time": 1}).
			Limit(100).All(&arr)

		data.Total = 100
		data.Value = arr
		if len(arr) > 0 {
			data.Open = arr[len(arr)-1].Close
			data.Close = arr[0].Close
		}
	}

	return data
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
			req.StartDate = "2012/01/01"
		case "w":
			req.StartDate = "2015/01/01"
		default:
			req.StartDate = "2019/01/01"
		}
	}

	t, _ := time.Parse("2006/01/02", req.StartDate)

	var data []bson.M
	db.KlineDB.Collection(util.Md5Code(req.Code)).Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"code": req.Code, "time": bson.M{"$gt": t}}).
		Group(bson.M{
			"_id":           bson.M{"$dateToString": bson.M{"format": format, "date": "$time"}},
			"time":          bson.M{"$last": "$time"},
			"open":          bson.M{"$first": "$open"},
			"close":         bson.M{"$last": "$close"},
			"high":          bson.M{"$max": "$high"},
			"low":           bson.M{"$min": "$low"},
			"main_net":      bson.M{"$sum": "$main_net"},
			"vol":           bson.M{"$sum": "$vol"},
			"amount":        bson.M{"$sum": "$amount"},
			"pct_chg":       bson.M{"$sum": "$pct_chg"},
			"tr":            bson.M{"$sum": "$tr"},
			"balance":       bson.M{"$last": "$balance"},
			"winner_rate":   bson.M{"$last": "$winner_rate"},
			"hold_ratio_cn": bson.M{"$last": "$hold_ratio_cn"},
			"net_vol_cn":    bson.M{"$sum": "$net_vol_cn"},
		}).
		Sort(bson.M{"time": 1}).Do()).All(&data)

	if req.Head > 0 {
		midware.Success(c, data[:req.Head])

	} else if req.Tail > 0 {
		midware.Success(c, data[len(data)-req.Tail:])

	} else {
		midware.Success(c, data)
	}
}
