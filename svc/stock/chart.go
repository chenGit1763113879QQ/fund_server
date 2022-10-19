package stock

import (
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
	}

	db.MinuteDB.Collection(job.GetMarket(code).ParseTradeDate()).
		Find(ctx, bson.M{"code": code}).All(&data)
	return data
}

func GetSimpleChart(code string, chartType string) any {
	var data struct {
		Total uint    `json:"total"`
		Value any     `json:"value"`
		Open  float64 `json:"open"`
	}

	switch chartType {
	case "minute":
		var arr []struct {
			Time   int64   `json:"time"`
			PctChg float64 `json:"value" bson:"pct_chg"`
		}

		m := job.GetMarket(code)
		if m == nil {
			return nil
		}

		db.MinuteDB.Collection(m.ParseTradeDate()).
			Find(ctx, bson.M{"code": code, "minute": bson.M{"$mod": bson.A{2, 0}}}).
			Sort("time").All(&arr)

		switch m.Market {
		case util.CN:
			data.Total = 240 / 2

		case util.HK:
			data.Total = 310 / 2

		case util.US:
			data.Total = 390 / 2
		}
		data.Value = arr

	case "60day":
		var arr []struct {
			Time  int64   `json:"time"`
			Close float64 `json:"value" bson:"close"`
		}

		t, _ := time.Parse("2006/01/02", "2022/01/01")

		db.KlineDB.Collection(util.Md5Code(code)).
			Find(ctx, bson.M{"code": code, "time": bson.M{"$gt": t.Unix()}}).
			Select(bson.M{"close": 1, "time": 1}).
			Sort("-time").Limit(60).All(&arr)

		data.Total = 60
		data.Value = arr
		if len(arr) > 0 {
			data.Open = arr[len(arr)-1].Close
		}
	}

	return data
}

func GetKline(c *gin.Context) {
	var req struct {
		Code      string    `form:"code" binding:"required"`
		Period    string    `form:"period" binding:"required"`
		StartDate time.Time `form:"start_date"`
		Head      int       `form:"head"`
		Tail      int       `form:"tail"`
	}
	if err := c.ShouldBind(&req); err != nil {
		midware.Error(c, err)
		return
	}

	format := map[string]string{
		"d": "%Y/%m/%d", "w": "%Y/%V", "m": "%Y/%m", "y": "%Y",
	}[req.Period]

	if req.StartDate.IsZero() {
		switch req.Period {
		case "y", "q", "m":
		case "w":
			req.StartDate, _ = time.Parse("2006/01/02", "2015/06/01")
		default:
			req.StartDate, _ = time.Parse("2006/01/02", "2020/01/01")
		}
	}

	fmtTime := bson.M{"$toDate": bson.M{"$multiply": bson.A{"$time", 1000}}}

	var data []bson.M
	db.KlineDB.Collection(util.Md5Code(req.Code)).Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"code": req.Code, "time": bson.M{"$gt": req.StartDate.Unix()}}).
		Group(bson.M{
			"_id":         bson.M{"$dateToString": bson.M{"format": format, "date": fmtTime}},
			"time":        bson.M{"$last": fmtTime},
			"open":        bson.M{"$first": "$open"},
			"close":       bson.M{"$last": "$close"},
			"high":        bson.M{"$max": "$high"},
			"low":         bson.M{"$min": "$low"},
			"main_net":    bson.M{"$sum": "$main_net"},
			"vol":         bson.M{"$sum": "$vol"},
			"amount":      bson.M{"$sum": "$amount"},
			"pct_chg":     bson.M{"$sum": "$pct_chg"},
			"balance":     bson.M{"$last": "$balance"},
			"winner_rate": bson.M{"$last": "$winner_rate"},
			"ratio":       bson.M{"$last": "$hold_ratio_cn"},
		}).Sort(bson.M{"time": 1}).
		Do()).All(&data)

	if req.Head > 0 {
		midware.Success(c, data[:req.Head])

	} else if req.Tail > 0 {
		midware.Success(c, data[len(data)-req.Tail:])

	} else {
		midware.Success(c, data)
	}
}
