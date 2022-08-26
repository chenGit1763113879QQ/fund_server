package pro

import (
	"fund/cache"
	"fund/db"
	"fund/model"
	"fund/util"
	"math"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
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

// 执行预测
func PredictStock() {
	db.Predict.DropCollection(ctx)

	p := util.NewPool(5)
	cache.Stock.RangeForCNStock(func(k string, v model.Stock) {
		// filter
		if v.Mc > 50*math.Pow(10, 8) {
			p.NewTask(func() {
				predict(k, 30)
			})
			p.NewTask(func() {
				predict(k, 60)
			})
		}
	})
	p.Wait()
}

// 预测算法
func predict(code string, days int) {
	src := klineMap.Load(code)
	if len(src) < days {
		return
	}

	// 矩阵转数组
	arr := closeHist(src)[len(src)-days:]
	arr = oneness(arr)

	// 结果集
	results := make([]map[string]any, 0)

	klineMap.Range(func(matchCode string, match []model.Kline) {
		if len(match) < days {
			return
		}

		dates := timeHist(match)
		closeLine := closeHist(match)

		// 移动窗口
		for i := 0; i+days+5 < len(match); i += 2 {
			// 行情矩阵
			mat := make([]float64, days)
			copy(mat, closeLine[i:i+days])

			mat = oneness(mat)
			res := std(arr, mat)

			if res < 0.25 {
				results = append(results, map[string]any{
					"预测股票": code, "匹配股票": matchCode,
					"匹配日期": dates[i], "匹配天数": days, "标准差": res / float64(days),
				})
			}
		}
	})

	// 保存标准差最小的五条数据
	res := dataframe.LoadMaps(results).Arrange(dataframe.Order{Colname: "标准差", Reverse: false})
	if res.Nrow() > 5 {
		db.Predict.InsertMany(ctx, res.Maps()[0:5])
	} else {
		db.Predict.InsertMany(ctx, res.Maps())
	}
}

// 归一化
func oneness(arr []float64) []float64 {
	factor := arr[0]
	for i := range arr {
		arr[i] /= factor
	}
	return arr
}

// 计算标准差
func std(arr1 []float64, arr2 []float64) float64 {
	for i := range arr2 {
		arr2[i] -= arr1[i]
	}
	return series.Floats(arr2).StdDev()
}
