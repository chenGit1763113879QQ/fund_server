package stock

import (
	"fund/db"
	"fund/midware"

	"cloud.google.com/go/civil"
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

	filter := bson.M{"logs.time": bson.M{"$gt": civil.Date{Year: 2017, Month: 9, Day: 1}}}
	if req.Code != "" {
		filter["code"] = req.Code
	}

	var data bson.M
	db.BackDB.Collection(req.Coll).Find(ctx, filter).Select(bson.M{"_id": 0}).One(&data)

	midware.Success(c, data)
}
