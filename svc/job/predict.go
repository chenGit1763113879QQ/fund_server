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

	"github.com/go-gota/gota/series"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

func PredictStock() {
	loadKlines()

	p := util.NewPool()
	for _, k := range getCNStocks() {
		p.NewTask(predict, k, "30")
		p.NewTask(predict, k, "60")
	}
	p.Wait()
	log.Debug().Msg("predict kline finished")
}

func loadKlines() {
	t, _ := time.Parse("2006/01/02", "2017/06/01")

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

			closeArr := make([]float64, len(data))
			timeArr := make([]time.Time, len(data))
			for i, k := range data {
				closeArr[i] = k.Close
				timeArr[i] = k.Time
			}
			cache.PreKlineMap.Store(id, closeArr, timeArr)

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
	srcClose, _ := cache.PreKlineMap.Load(code)
	if len(srcClose) < days {
		return
	}
	srcClose = srcClose[len(srcClose)-days:]
	oneness(srcClose)

	// 趋势
	trend := srcClose[0] > srcClose[len(srcClose)-1]

	// results
	bulk := db.Predict.Bulk()

	cache.PreKlineMap.Range(func(matchCode string, match []float64, times []time.Time) {
		if len(match) < days {
			return
		}

		// rolling window
		for i := 0; i+days+5 < len(match); i++ {
			// 判断趋势是否相同
			if tr := match[i] > match[i+days]; tr != trend {
				continue
			}

			// match
			matchClose := make([]float64, days)
			copy(matchClose, match[i:i+days])

			oneness(matchClose)
			res := std(srcClose, matchClose)

			if res < 0.01 {
				bulk.InsertOne(bson.M{
					"src_code":   code,
					"match_code": matchCode,
					"start_date": times[i],
					"end_date":   times[i+days],
					"period":     days,
					"std":        res,
				})
			}
		}
	})
	bulk.RemoveAll(bson.M{"src_code": code, "period": days})
	bulk.Run(ctx)

	db.LimitDB.Set(ctx, fmt.Sprintf("predict_%d:%s", days, code), "1", time.Hour*12)
}

func oneness(arr []float64) {
	factor := arr[0]
	for i := range arr {
		arr[i] /= factor
	}
}

func std(arr1 []float64, arr2 []float64) float64 {
	if len(arr1) != len(arr2) {
		log.Error().Msg("std error: len(arr1) != len(arr2)")
		return 999
	}
	for i := range arr2 {
		arr2[i] -= arr1[i]
	}
	return series.Floats(arr2).StdDev()
}
