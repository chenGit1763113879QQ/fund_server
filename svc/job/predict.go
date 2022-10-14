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

	"cloud.google.com/go/civil"
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
	t := civil.Date{Year: 2015, Month: 1, Day: 1}

	p := util.NewPool()
	for _, code := range getCNStocks() {
		p.NewTask(func(strs ...string) {
			id := strs[0]
			var data []*model.Kline

			// get kline
			db.KlineDB.Collection(util.Md5Code(id)).Aggregate(ctx, mongox.Pipeline().
				Match(bson.M{"code": id, "time": bson.M{"$gt": t}}).Do()).One(&data)

			cache.Kline.Store(id, data)
		}, code)
	}
	p.Wait()

	log.Info().Msgf("init predict kline[%d] success", cache.Kline.Len())
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
	df := cache.Kline.LoadPKline(code)
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

	cache.Kline.RangePKline(func(k string, v *model.PreKline) {
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
					MatchCode: k,
					StartDate: v.Time[i],
					Period:    days,
					Limit:     days + PREDICT_DAYS,
					Std:       stdSum,
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
