package art

import (
	"context"
	"fund/db"
	"fund/midware"
	"fund/util/mongox"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	pr "go.mongodb.org/mongo-driver/bson/primitive"
)

const pageSize = 15

var ctx = context.Background()

func getUserArtId(c *gin.Context) (uid pr.ObjectID, aid pr.ObjectID) {
	uid = c.MustGet("id").(pr.ObjectID)
	aid, _ = pr.ObjectIDFromHex(c.Param("id"))

	return uid, aid
}

// 获取文章信息
func GetArticle(c *gin.Context) {
	uid, aid := getUserArtId(c)

	// 阅读
	db.Article.UpdateId(ctx, aid, bson.M{"$addToSet": bson.M{"reads": uid}})

	data := bson.M{}
	db.Article.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"_id": aid}).
		Lookup("user", "uid", "_id", "user").
		Lookup("user", "likes", "_id", "likeList").
		Lookup("user", "colls", "_id", "collList").
		Lookup("stock", "tag", "_id", "stock").
		Unwind("$user").
		Project(bson.M{
			"title": 1, "content": 1, "type": 1, "createAt": 1, "updateAt": 1,
			"isMine":   bson.M{"$eq": bson.A{uid, "$uid"}},
			"inLike":   bson.M{"$in": bson.A{uid, "$likes"}},
			"likes":    bson.M{"$size": "$likes"},
			"likeList": bson.M{"_id": 1, "name": 1},

			"inColl":   bson.M{"$in": bson.A{uid, "$colls"}},
			"colls":    bson.M{"$size": "$colls"},
			"collList": bson.M{"_id": 1, "name": 1},

			"comments": 1,
			"user":     bson.M{"_id": 1, "name": 1, "ip.region": 1},
			"stock":    bson.M{"_id": 1, "cid": 1, "name": 1, "marketType": 1, "type": 1, "price": 1, "pct_chg": 1},
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

	uid := c.MustGet("id").(pr.ObjectID)

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
		Lookup("user", "uid", "_id", "user").
		Unwind("$user").
		Project(bson.M{
			"title": 1, "content": 1, "type": 1, "createAt": 1, "updateAt": 1,
			"inLike":   bson.M{"$in": bson.A{uid, "$likes"}},
			"likes":    bson.M{"$size": "$likes"},
			"user._id": 1, "user.name": 1, "comments": 1,
		}).Do()).All(&data)

	midware.Auto(c, err, data)
}
