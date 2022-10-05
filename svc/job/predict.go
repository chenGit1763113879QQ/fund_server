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
	"go.mongodb.org/mongo-driver/bson"
	"gonum.org/v1/gonum/stat"
)

const PREDICT_DAYS = 20 // 展示的预测天数

func PredictStock() {
	loadKlines()

	p := util.NewPool()
	for _, k := range getCNStocks() {
		p.NewTask(predict, k, "60")
	}
	p.Wait()
	log.Debug().Msg("predict kline finished")
}

func loadKlines() {
	t, _ := time.Parse("2006/01/02", "2013/01/01")

	p := util.NewPool()
	for _, code := range getCNStocks() {
		p.NewTask(func(strs ...string) {
			id := strs[0]
			var data []*model.Kline

			// get kline
			db.KlineDB.Collection(util.Md5Code(id)).Aggregate(ctx, mongox.Pipeline().
				Match(bson.M{"code": id, "time": bson.M{"$gt": t}}).
				Do()).All(&data)

			priceArr := make([]float64, len(data))
			timeArr := make([]time.Time, len(data))
			for i, k := range data {
				priceArr[i] = k.WinnerRate - 50
				timeArr[i] = k.Time
			}
			cache.PreKlineMap.Store(id, priceArr, timeArr)
		}, code)
	}
	p.Wait()

	log.Info().Msgf("init predict kline[%d] success", cache.PreKlineMap.Len())
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
	src, _ := cache.PreKlineMap.Load(code)
	if len(src) < days {
		return
	}
	src = src[len(src)-days:]

	// trend
	trend := src[0] > src[days-1]

	// results
	db.Predict.RemoveAll(ctx, bson.M{"src_code": code, "period": days})
	results := make(model.PredictArr, 0)

	cache.PreKlineMap.Range(func(matchCode string, match []float64, times []time.Time) {
		if len(match) < days {
			return
		}
		// rolling window
		for i := 0; i+days+PREDICT_DAYS < len(match); i++ {
			// trend
			if trend != (match[i] > match[i+days-1]) {
				continue
			}
			results = append(results, &model.PredictRes{
				SrcCode:   code,
				MatchCode: matchCode,
				StartDate: times[i],
				Period:    days,
				Limit:     days + PREDICT_DAYS,
				Std:       std(src, match[i:i+days]),
			})
		}
	})

	sort.Sort(results)
	if results.Len() > 20 {
		results = results[0:20]
	}

	// filter closely date
	for i := 0; i < results.Len()-2; i++ {
		l := results[i]
		r := results[i+1]

		if l == nil || r == nil {
			continue
		}
		if r.StartDate.Sub(l.StartDate) <= time.Hour*72 {
			if l.Std > r.Std {
				results[i] = nil
			} else {
				results[i+1] = nil
			}
		}
	}
	for i, p := range results {
		if p == nil {
			results = append(results[:i], results[i+1:]...)
		}
	}

	// save db
	db.Predict.InsertMany(ctx, results)
	db.LimitDB.Set(ctx, fmt.Sprintf("predict_%d:%s", days, code), "1", time.Hour*12)
}

func oneness(arr []float64, factors ...float64) {
	factor := arr[0]
	if len(factors) > 0 {
		factor = factors[0]
	}
	for i := range arr {
		arr[i] /= (factor / 100)
	}
}

func std(arr1 []float64, arr2 []float64) float64 {
	arr := make([]float64, len(arr1))
	for i := range arr {
		arr[i] = arr1[i] - arr2[i]
	}
	return stat.StdDev(arr, nil)
}
