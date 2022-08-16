package art

import (
	"context"
	"fund/db"
	"fund/midware"
	"fund/model"
	"fund/util/mongox"
	"time"

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

// 热门文章
func GetHots(c *gin.Context) {
	req := new(model.ArtOpt)
	c.ShouldBind(req)
	req.Uid = c.MustGet("id").(pr.ObjectID)

	var arts []bson.M

	filter := bson.M{}
	if req.Code != "" {
		filter["tag"] = req.Code
	} else {
		filter["createAt"] = bson.M{"$gt": time.Now().AddDate(0, 0, -7)}
	}

	db.Article.Aggregate(ctx, mongox.Pipeline().
		Match(filter).
		Lookup("user", "uid", "_id", "user").
		Unwind("$user").
		Project(bson.M{
			"title": 1, "content": 1, "createAt": 1,
			// 计算实时热度
			"hot": bson.M{"$divide": bson.A{
				bson.M{"$add": bson.A{
					bson.M{"$size": "$reads"},
					bson.M{"$multiply": bson.A{2, bson.M{"$size": "$likes"}}},
					bson.M{"$multiply": bson.A{3, bson.M{"$size": "$colls"}}},
					bson.M{"$multiply": bson.A{4, "$comments"}},
				}},
				bson.M{"$subtract": bson.A{time.Now(), "$createAt"}},
			}},
			"user": bson.M{"_id": 1, "name": 1},
		}).
		Sort(bson.M{"hot": -1, "createAt": -1}).
		Skip(pageSize*(req.Page-1)).
		Limit(pageSize).
		Do()).All(&arts)

	midware.Success(c, arts)
}

// 新建文章
func NewArticle(c *gin.Context) {
	req := new(model.Article)
	c.ShouldBind(req)
	req.Uid = c.MustGet("id").(pr.ObjectID)

	req.Reads = []pr.ObjectID{}
	req.Likes = []pr.ObjectID{}
	req.Colls = []pr.ObjectID{}
	req.CreateAt = time.Now()

	_, err := db.Article.InsertOne(ctx, req)
	midware.Auto(c, err, nil, "发布成功")
}

// 删除文章
func DelArticle(c *gin.Context) {
	uid, aid := getUserArtId(c)

	err := db.Article.Remove(ctx, bson.M{"uid": uid, "_id": aid})
	midware.Auto(c, err, nil, "删除成功")
}

// 我的文章
func MyArticles(c *gin.Context) {
	uid := c.MustGet("id").(pr.ObjectID)
	arts := make([]bson.M, 0)

	err := db.Article.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"uid": uid}).
		Lookup("user", "uid", "_id", "user").
		Unwind("$user").
		Project(bson.M{
			"title": 1, "content": 1, "createAt": 1, "updateAt": 1,
			"inLike":   bson.M{"$in": bson.A{uid, "$likes"}},
			"likes":    bson.M{"$size": "$likes"},
			"user._id": 1, "user.name": 1, "comments": 1,
		}).
		Sort(bson.M{"createAt": -1}).
		Do()).All(&arts)

	midware.Auto(c, err, arts)
}
