package job

import (
	"fund/db"
	"fund/model"
	"fund/util"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func getNews() {
	var stocks []struct {
		Code string `bson:"_id"`
		Name string `bson:"name"`
	}
	var news []struct {
		Datetime string `csv:"datetime"`
		Content  string `csv:"content"`
		Title    string `csv:"title"`
	}

	// wait
	time.Sleep(time.Second * 5)
	db.Stock.Find(ctx, bson.M{}).All(&stocks)

	// 去除多余后缀
	for i := range stocks {
		for _, tail := range []string{"-SW", "-W", "-S", "-U", "-WD"} {
			stocks[i].Name = strings.Split(stocks[i].Name, tail)[0]
		}
	}
	location, _ := time.LoadLocation("Asia/Shanghai")

	for {
		err := util.TushareApi("news", bson.M{"src": "eastmoney"}, "datetime,title,content", &news)
		if err != nil {
			return
		}
		for _, n := range news {
			// 去除【行情】类资讯
			if strings.Contains(n.Content, "【行情】") {
				continue
			}
			// 匹配
			codes := make([]string, 0)
			for _, s := range stocks {
				if strings.Contains(n.Title, s.Name) && s.Name != "证券" {
					if len(codes) < 3 {
						codes = append(codes, s.Code)
					}
				}
			}
			t, _ := time.ParseInLocation("2006-01-02 15:04:05", n.Datetime, location)
			db.Article.InsertOne(ctx, &model.Article{
				Title:    n.Title,
				Content:  n.Content,
				CreateAt: t,
				Tag:      codes,
			})
		}
		time.Sleep(time.Minute)
	}
}
