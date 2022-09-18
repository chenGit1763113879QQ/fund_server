package job

import (
	"fund/db"
	"fund/model"
	"fund/util"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

func getNews() {
	var stocks []*struct {
		Code string `bson:"_id"`
		Name string `bson:"name"`
	}
	db.Stock.Find(ctx, bson.M{}).All(&stocks)

	var news []*struct {
		Datetime string `csv:"datetime"`
		Content  string `csv:"content"`
		Title    string `csv:"title"`
	}

	// 中文名去除后缀
	for _, s := range stocks {
		if util.IsChinese(s.Name) {
			s.Name = strings.Split(s.Name, "-")[0]
		}
	}

	loc, _ := time.LoadLocation("Asia/Shanghai")

	if err := util.TushareApi("news", bson.M{"src": "eastmoney"}, "datetime,title,content", &news); err != nil {
		log.Error().Msg(err.Error())
		return
	}

	for _, n := range news {
		// 匹配
		codes := make([]string, 0)
		for _, s := range stocks {
			if strings.Contains(n.Title, s.Name) && s.Name != "证券" {
				if len(codes) < 5 {
					codes = append(codes, s.Code)
				}
			}
		}

		t, _ := time.ParseInLocation("2006-01-02 15:04:05", n.Datetime, loc)
		db.Article.InsertOne(ctx, &model.Article{
			Title:    n.Title,
			Content:  n.Content,
			CreateAt: t,
			Tag:      codes,
		})
	}
}
