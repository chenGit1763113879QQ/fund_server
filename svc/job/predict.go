package job

import (
	"fmt"
	"fund/cache"
	"fund/db"
	"fund/model"
	"fund/util"
	"fund/util/mongox"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"gonum.org/v1/gonum/stat"
)

func PredictStock() {
	loadKlines()

	p := util.NewPool()
	for _, k := range getCNStocks() {
		// p.NewTask(predict, k, "30")
		p.NewTask(predict, k, "60")
	}
	p.Wait()
	log.Debug().Msg("predict kline finished")
}

func loadKlines() {
	t, _ := time.Parse("2006/01/02", "2016/01/01")

	p := util.NewPool()
	for _, code := range getCNStocks() {
		p.NewTask(func(strs ...string) {
			id := strs[0]
			var data []*model.Kline

			// get kline
			db.KlineDB.Collection(util.Md5Code(id)).Aggregate(ctx, mongox.Pipeline().
				Match(bson.M{"code": id, "time": bson.M{"$gt": t}}).
				Sort(bson.M{"time": 1}).Do()).
				All(&data)

			priceArr := make([]float64, len(data))
			timeArr := make([]time.Time, len(data))
			for i, k := range data {
				priceArr[i] = k.WinnerRate
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

	// results
	db.Predict.RemoveAll(ctx, bson.M{"src_code": code, "period": days})
	results := make([]bson.M, 0)

	cache.PreKlineMap.Range(func(matchCode string, match []float64, times []time.Time) {
		if len(match) < days {
			return
		}

		// rolling window
		for i := 0; i+days+20 < len(match); i++ {
			res := std(src, match[i:i+days])

			t := bson.M{
				"src_code":   code,
				"match_code": matchCode,
				"start_date": times[i].Format("2006/01/02"),
				"period":     days,
				"limit":      days + 20,
				"std":        res,
			}
			if len(results) < 50 {
				results = append(results, t)

			} else {
				// 替换最大的标准差
				max, index := res, -1

				for i, item := range results {
					if item["std"].(float64) > max {
						max = item["std"].(float64)
						index = i
					}
				}
				if index >= 0 {
					results[index] = t
				}
			}
		}
	})
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
		arr[i] /= factor
		arr[i] *= 100
	}
}

func std(arr1 []float64, arr2 []float64) float64 {
	if len(arr1) != len(arr2) {
		panic("arr1 != arr2")
	}
	arr := make([]float64, len(arr1))
	for i := range arr {
		arr[i] = arr1[i] - arr2[i]
	}
	return stat.StdDev(arr, nil)
}
