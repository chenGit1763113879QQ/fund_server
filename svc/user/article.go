package user

import (
	"fund/db"
	"fund/midware"
	"fund/util/mongox"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

const pageSize = 15

// 获取文章信息
func GetArticle(c *gin.Context) {
	aid := c.Param("id")

	data := bson.M{}
	db.Article.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"_id": aid}).
		Lookup("stock", "tag", "_id", "stock").
		Project(bson.M{
			"title": 1, "content": 1, "createAt": 1,
			"stock": bson.M{"_id": 1, "cid": 1, "name": 1, "marketType": 1, "type": 1, "price": 1, "pct_chg": 1},
		}).Do()).One(&data)

	// add 60day
	// stk, _ := data["stock"].(bson.A)
	// stocks := make([]bson.M, 0)
	// for _, i := range stk {
	// 	stocks = append(stocks, i.(bson.M))
	// }

	// stock.AddChart("60day", stocks)
	// data["stock"] = stocks

	midware.Success(c, data)
}

// 资讯文章
func GetNews(c *gin.Context) {
	var req struct {
		Code string `form:"code"`
		Page int    `form:"page" binding:"required,min=1"`
	}
	if err := c.ShouldBind(&req); err != nil {
		midware.Warning(c, err.Error())
		return
	}

	var data []bson.M

	filter := bson.M{"type": 2}
	if req.Code != "" {
		filter["tag"] = req.Code
	}

	err := db.Article.Aggregate(ctx, mongox.Pipeline().
		Match(filter).
		Sort(bson.M{"createAt": -1}).
		Skip(pageSize*(req.Page-1)).
		Limit(pageSize).
		Project(bson.M{
			"title": 1, "content": 1, "createAt": 1,
			"user._id": 1, "user.name": 1, "comments": 1,
		}).Do()).All(&data)

	midware.Auto(c, err, data)
}
