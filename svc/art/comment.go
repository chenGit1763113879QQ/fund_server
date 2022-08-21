package art

import (
	"fund/db"
	"fund/midware"
	"fund/model"
	"fund/util/mongox"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	pr "go.mongodb.org/mongo-driver/bson/primitive"
)

// 获取评论
func GetComments(c *gin.Context) {
	aid, _ := pr.ObjectIDFromHex(c.Param("id"))
	data := make([]bson.M, 0)

	err := db.Comment.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"aid": aid}).
		Lookup("user", "uid", "_id", "user").
		Unwind("$user").
		Project(bson.M{
			"content": 1, "type": 1, "createAt": 1,
			"user._id": 1, "user.name": 1, "likes": 1,
		}).
		Sort(bson.M{"createAt": -1}).
		Limit(20).Do()).All(&data)

	midware.Auto(c, err, data)
}

// 发表评论
func WriteComment(c *gin.Context) {
	req := new(model.Comment)
	req.Uid, req.Aid = getUserArtId(c)
	req.CreateAt = time.Now()

	c.ShouldBind(req)
	

	if _, err := db.Comment.InsertOne(ctx, req); err != nil {
		midware.Error(c, err)
	}
	err := db.Article.UpdateId(ctx, req.Aid, bson.M{"$inc": bson.M{"comments": 1}})

	midware.Auto(c, err, nil, "评论成功")
}

// 删除评论
func DelComment(c *gin.Context) {
	uid, cid := getUserArtId(c)

	err := db.Comment.Remove(ctx, bson.M{"_id": cid, "uid": uid})
	if err != nil {
		midware.Error(c, err)
	}

	midware.Auto(c, err, nil, "评论已删除")
}

// 点赞
func Like(c *gin.Context) {
	uid, aid := getUserArtId(c)
	err := db.Article.UpdateId(ctx, aid, bson.M{"$addToSet": bson.M{"likes": uid}})

	midware.Auto(c, err, nil)
}

// 取消点赞
func UnLike(c *gin.Context) {
	uid, aid := getUserArtId(c)
	err := db.Article.UpdateId(ctx, aid, bson.M{"$pull": bson.M{"likes": uid}})

	midware.Auto(c, err, nil)
}

// 收藏
func Collect(c *gin.Context) {
	uid, aid := getUserArtId(c)

	if err := db.Article.UpdateId(ctx, aid, bson.M{"$addToSet": bson.M{"colls": uid}}); err != nil {
		midware.Error(c, err)
	}
	err := db.User.UpdateId(ctx, uid, bson.M{"$addToSet": bson.M{"colls": aid}})

	midware.Auto(c, err, nil)
}

// 取消收藏
func UnCollect(c *gin.Context) {
	uid, aid := getUserArtId(c)

	if err := db.Article.UpdateId(ctx, aid, bson.M{"$pull": bson.M{"colls": uid}}); err != nil {
		midware.Error(c, err)
	}
	err := db.User.UpdateId(ctx, uid, bson.M{"$pull": bson.M{"colls": aid}})

	midware.Auto(c, err, nil)
}

// 获取收藏列表
func CollList(c *gin.Context) {
	uid := c.MustGet("id").(pr.ObjectID)

	data := make([]bson.M, 0)
	err := db.User.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"_id": uid}).
		Lookup("article", "colls", "_id", "arts").
		Project(bson.M{
			"_id": 0, "arts": bson.M{
				"_id": 1, "comments": 1, "createAt": 1, "title": 1, "likes": 1,
			},
		}).
		Do()).One(&data)

	midware.Auto(c, err, data)
}
