package job

import (
	"fmt"
	"fund/cache"
	"fund/db"
	"fund/model"
	"time"

	"github.com/go-gota/gota/series"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

func PredictStock() {
	for _, k := range getCNStocks() {
		fmt.Println(k)
		predict(k, 30)
		predict(k, 60)
	}
	log.Debug().Msg("predict kline finished")
}

func predict(code string, days int) {
	// cache
	exist, _ := db.LimitDB.Exists(ctx, fmt.Sprintf("predict_%d:%s", days, code)).Result()
	if exist > 0 {
		return
	}

	src := cache.KlineMap.Load(code)
	if len(src) < days {
		return
	}

	var matchClose []float64

	// oneness src
	srcClose := closeIndex(src, len(src)-days, len(src))
	oneness(srcClose)

	// results
	bulk := db.Predict.Bulk()

	cache.KlineMap.Range(func(matchCode string, match []*model.Kline) {
		if len(match) < days {
			return
		}
		// rolling window
		for i := 0; i+days+5 < len(match); i++ {

			// oneness match
			matchClose = closeIndex(match, i, i+days)
			oneness(matchClose)

			// cal std
			res := std(srcClose, matchClose)

			if res <= 0.01 {
				bulk.InsertOne(bson.M{
					"src_code": code, "match_code": matchCode,
					"match_date": match[i].Time, "period": days, "std": res,
				})
			}
		}
	})
	bulk.Remove(bson.M{"src_code": code, "period": days})
	bulk.Run(ctx)

	db.LimitDB.Set(ctx, fmt.Sprintf("predict_%d:%s", days, code), "1", time.Hour*12)
}

func closeIndex(k []*model.Kline, start int, end int) []float64 {
	arr := make([]float64, end-start)
	for i := start; i < end; i++ {
		arr[i-start] = k[i].Close
	}
	return arr
}

func oneness(arr []float64) {
	factor := arr[0]
	for i := range arr {
		arr[i] /= factor
	}
}

func std(arr1 []float64, arr2 []float64) float64 {
	for i := range arr2 {
		arr2[i] -= arr1[i]
	}
	return series.Floats(arr2).StdDev()
}
