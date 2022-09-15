package pro

import (
	"fund/db"
	"fund/model"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/rs/zerolog/log"
)

func timeHist(k []model.Kline) []time.Time {
	arr := make([]time.Time, len(k))
	for i := range k {
		arr[i] = k[i].Time
	}
	return arr
}

func closeHist(k []model.Kline) []float64 {
	arr := make([]float64, len(k))
	for i := range k {
		arr[i] = k[i].Close
	}
	return arr
}

func PredictStock() {
	db.Predict.DropCollection(ctx)

	for _, k := range getCNStocks() {
		predict(k, 30)
		predict(k, 60)
	}
	log.Debug().Msg("predict kline finished")
}

func predict(code string, days int) {
	src := klineMap.Load(code)
	if len(src) < days {
		return
	}

	// matrix to array
	arr := closeHist(src)[len(src)-days:]
	oneness(arr)

	// results
	results := make([]map[string]any, 0)

	klineMap.Range(func(matchCode string, match []model.Kline) {
		if len(match) < days {
			return
		}

		dates := timeHist(match)
		closeLine := closeHist(match)

		// rolling window
		for i := 0; i+days+5 < len(match); i++ {

			// matrix
			mat := make([]float64, days)
			copy(mat, closeLine[i:i+days])

			oneness(mat)
			res := std(arr, mat)

			if res < 0.25 {
				results = append(results, map[string]any{
					"p_code": code, "m_code": matchCode,
					"m_date": dates[i], "m_period": days, "std": res / float64(days),
				})
			}
		}
	})

	res := dataframe.LoadMaps(results).Arrange(dataframe.Order{Colname: "std", Reverse: false})
	if res.Nrow() > 5 {
		db.Predict.InsertMany(ctx, res.Maps()[0:5])
	} else {
		db.Predict.InsertMany(ctx, res.Maps())
	}
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
