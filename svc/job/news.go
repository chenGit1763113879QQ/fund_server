package job

import (
	"fund/db"
	"fund/model"
	"fund/util"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	pr "go.mongodb.org/mongo-driver/bson/primitive"
)

var NewsID pr.ObjectID

func init() {
	db.User.InsertOne(ctx, &model.User{
		Email:    "lucario@example.com",
		Name:     "7*24资讯",
		Password: "w&vme@.W=dd_wAdv",
	})

	user := new(model.User)
	db.User.Find(ctx, bson.M{"name": "7*24资讯"}).One(&user)
	NewsID = user.Id
}

// 获取新闻资讯
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

	// 等待初始化
	time.Sleep(time.Second * 3)
	db.Stock.Find(ctx, bson.M{}).All(&stocks)

	// 去除多余后缀
	for i := range stocks {
		for _, tail := range []string{"-SW", "-W", "-S", "-U", "-WD"} {
			stocks[i].Name = strings.Split(stocks[i].Name, tail)[0]
		}
	}
	// 时区
	location, _ := time.LoadLocation("Asia/Shanghai")

	// 下载函数
	getData := func() {
		params := bson.M{"src": "eastmoney"}

		if err := util.TushareApi("news", params, "datetime,title,content", &news); err != nil {
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
					codes = append(codes, s.Code)
				}
			}
			if len(codes) > 3 {
				codes = codes[0:3]
			}
			t, _ := time.ParseInLocation("2006-01-02 15:04:05", n.Datetime, location)
			db.Article.InsertOne(ctx, &model.Article{
				Uid:      NewsID,
				Title:    n.Title,
				Content:  n.Content,
				Reads:    make([]pr.ObjectID, 0),
				Likes:    make([]pr.ObjectID, 0),
				Colls:    make([]pr.ObjectID, 0),
				CreateAt: t,
				Type:     2,
				Tag:      codes,
			})
		}

	}
	for {
		getData()
		time.Sleep(time.Minute)
	}
}
