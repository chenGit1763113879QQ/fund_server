package job

import (
	"fmt"
	"fund/cache"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"sort"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/xgzlucario/structx"
	"go.mongodb.org/mongo-driver/bson"
	"gonum.org/v1/gonum/stat"
)

const PREDICT_DAYS = 20 // 展示的预测天数

func PredictStock() {
	p := structx.NewPool[string]()
	for _, k := range getCNStocks() {
		p.NewTask(predict, k, "60")
	}
	p.Wait()
	log.Debug().Msg("predict kline finished")
}

func loadKline() {
	t, _ := time.Parse("2006/01/02", "2015/01/01")
	cache.New(getCNStocks())

	p := structx.NewPool[string]()
	for _, code := range getCNStocks() {
		p.NewTask(func(strs ...string) {
			id := strs[0]
			var data []*model.Kline

			// get kline
			db.KlineDB.Collection(util.Md5Code(id)).Aggregate(ctx, mongox.Pipeline().
				Match(bson.M{"code": id, "time": bson.M{"$gt": t}}).
				Do()).All(&data)

			cache.Store(id, data)
		}, code)
	}
	p.Wait()

	log.Info().Msgf("init predict kline[%d] success", cache.Len())
}

func predict(strs ...string) {
	code := strs[0]
	days, _ := strconv.Atoi(strs[1])

	// cache
	exist, _ := db.LimitDB.Exists(ctx, fmt.Sprintf("predict_%d:%s", days, code)).Result()
	if exist > 0 {
		return
	}

	// src
	df := cache.LoadPKline(code)
	n1 := df.Len()
	if n1 < days {
		return
	}

	df.Close = oneness(df.Close[n1-days:])
	// trend
	trend := df.Close[0] > df.Close[len(df.Close)-1]

	// results
	db.Predict.RemoveAll(ctx, &model.PredictRes{SrcCode: code, Period: days})
	results := make(model.PredictArr, 0)

	cache.RangePKline(func(k string, v *model.PreKline) {
		res := &model.PredictRes{
			SrcCode:   code,
			MatchCode: k,
			Period:    days,
			Limit:     days + PREDICT_DAYS,
			Std:       999,
		}

		// rolling window
		for lp := 0; lp < v.Len()-PREDICT_DAYS-days; lp++ {
			rp := lp + days
			// trend
			if (v.Close[lp] > v.Close[rp]) != trend {
				continue
			}
			// std
			stdSum := std(df.Close, oneness(v.Close[lp:rp]))

			if stdSum < res.Std {
				res.StartDate = time.Unix(v.Time[lp], 0)
				res.Std = stdSum
				res.PrePctChg = (v.Close[rp]/v.Close[lp] - 1) * 100
			}
		}
		results = append(results, res)
	})

	sort.Sort(results)
	results = results[0:20]

	// save db
	db.Predict.InsertMany(ctx, results)
	db.LimitDB.Set(ctx, fmt.Sprintf("predict_%d:%s", days, code), 1, time.Hour*12)
}

func oneness(arr []float64, factors ...float64) []float64 {
	factor := arr[0]
	if len(factors) > 0 {
		factor = factors[0]
	}
	newArr := make([]float64, len(arr))
	for i := range newArr {
		newArr[i] = arr[i] / factor * 100
	}
	return newArr
}

func std(arr1 []float64, arr2 []float64) float64 {
	arr := make([]float64, len(arr1))
	for i := range arr {
		arr[i] = arr1[i] - arr2[i]
	}
	return stat.StdDev(arr, nil)
}
