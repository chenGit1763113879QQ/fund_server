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
	t, _ := time.Parse("2006/01/02", "2017/01/01")

	p := util.NewPool()
	for _, code := range getCNStocks() {
		p.NewTask(func(strs ...string) {
			id := strs[0]
			var data cache.PreData

			// get kline
			db.KlineDB.Collection(util.Md5Code(id)).Aggregate(ctx, mongox.Pipeline().
				Match(bson.M{"code": id, "time": bson.M{"$gt": t}}).
				Group(bson.M{
					"_id":   1,
					"time":  bson.M{"$push": "$time"},
					"open":  bson.M{"$push": "$open"},
					"close": bson.M{"$push": "$close"},
				}).Do()).One(&data)

			cache.PreKlineMap.Store(id, data)
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
	df := cache.PreKlineMap.Load(code)
	n := df.Len()
	if n < days {
		return
	}
	factor := df.Open[n-days]
	df.Open = oneness(df.Open[n-days:], factor)
	df.Close = oneness(df.Close[n-days:], factor)

	// results
	db.Predict.RemoveAll(ctx, &model.PredictRes{SrcCode: code, Period: days})
	results := make(model.PredictArr, 0)

	cache.PreKlineMap.Range(func(key string, v cache.PreData) {
		if v.Len() < days {
			return
		}
		// rolling window
		for i := 0; i+days+PREDICT_DAYS < v.Len(); i++ {
			factor := v.Open[i]
			stdSum := util.Sum(
				std(df.Open, oneness(v.Open[i:i+days], factor)),
				std(df.Close, oneness(v.Close[i:i+days], factor)),
			)
			if stdSum < 8 {
				i++
				results = append(results, &model.PredictRes{
					SrcCode:   code,
					MatchCode: key,
					StartDate: v.Time[i],
					Period:    days,
					Limit:     days + PREDICT_DAYS,
					Std:       stdSum,
					PreDirect: v.Close[i+days] > v.Close[i],
					PrePctChg: (v.Close[i+days]/v.Close[i] - 1) * 100,
				})
			}
		}
	})

	sort.Sort(results)
	if results.Len() > 10 {
		results = results[0:10]
	}
	// save db
	db.Predict.InsertMany(ctx, results)
	db.LimitDB.Set(ctx, fmt.Sprintf("predict_%d:%s", days, code), "1", time.Hour*12)
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
