package stock

import (
	"fund/db"
	"fund/midware"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func GetBackLogs(c *gin.Context) {
	var req struct {
		Coll string `form:"coll" binding:"required"`
		Code string `form:"code"`
	}
	if err := c.ShouldBind(&req); err != nil {
		midware.Error(c, err)
	}

	t, _ := time.Parse("2006/01/02", "2018/01/01")
	filter := bson.M{"logs.time": bson.M{"$gt": t}}
	if req.Code != "" {
		filter["code"] = req.Code
	}

	var data bson.M
	db.BackDB.Collection(req.Coll).Find(ctx, filter).Select(bson.M{"_id": 0}).One(&data)

	midware.Success(c, data)
}
