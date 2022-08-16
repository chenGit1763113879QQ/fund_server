package pro

import (
	"fund/db"
	"fund/util"
	"fund/util/pool"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var klineCache map[string]dataframe.DataFrame

// 执行预测
func PredictStock() {
	db.Predict.DropCollection(ctx)
	items := initStock()

	// 加载缓存
	klineCache = make(map[string]dataframe.DataFrame)
	for _, i := range items {
		df := getKline(i.Id)
		// 验证合法
		if df.Nrow() >= 30 {
			klineCache[i.Id] = df
		}
	}

	p := pool.NewPool(5)
	for i := range items {
		p.NewTask(func() {
			predict(items[i].Id, 30)
		})
		p.NewTask(func() {
			predict(items[i].Id, 60)
		})
	}
	p.Wait()
}

// 预测算法
func predict(code string, days int) {
	src := klineCache[code]
	if src.Nrow() < days {
		return
	}

	// 矩阵转数组
	arr := src.Col("close").Float()[src.Nrow()-days:]
	arr = oneness(arr)

	// 结果集
	results := make([]map[string]any, 0)

	for matchCode, match := range klineCache {
		if match.Nrow() < days {
			continue
		}
		dates := match.Col("time").Records()
		closeLine := match.Col("close").Float()

		// 移动窗口
		for i := 0; i+days+5 < match.Nrow(); i += 2 {
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
	}
	// 保存标准差最小的五条数据
	res := dataframe.LoadMaps(results).Arrange(dataframe.Order{Colname: "标准差", Reverse: false})
	if res.Nrow() > 5 {
		db.Predict.InsertMany(ctx, res.Maps()[0:5])
	} else {
		db.Predict.InsertMany(ctx, res.Maps())
	}

}

// 获取k线数据
func getKline(code string) dataframe.DataFrame {
	var kline []map[string]interface{}

	t, _ := time.Parse("2006-01-02", "2017-06-01")
	db.KlineDB.Collection(util.CodeToInt(code)).Aggregate(ctx, mongo.Pipeline{
		bson.D{{"$match", bson.M{"code": code, "time": bson.M{"$gt": t}}}},
		bson.D{{"$project", bson.M{
			"time": bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$time"}},
			"_id":  0, "close": 1,
		}}},
		bson.D{{"$sort", bson.M{"time": 1}}},
	}).All(&kline)

	return dataframe.LoadMaps(kline)
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
