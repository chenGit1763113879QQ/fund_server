package stock

import (
	"errors"
	"fund/db"
	"fund/midware"
	"fund/model"
	"fund/util"
	"fund/util/mongox"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	pr "go.mongodb.org/mongo-driver/bson/primitive"
)

func getGroups(id pr.ObjectID) *model.Groups {
	g := new(model.Groups)
	db.User.Find(ctx, bson.M{"_id": id}).One(g)
	return g
}

// 获取分组行情
func GetGroups(c *gin.Context) {
	uid := c.MustGet("id").(pr.ObjectID)
	chart := c.Query("chart")

	var data model.Groups
	db.User.Aggregate(ctx, mongox.Pipeline().
		Match(bson.M{"_id": uid}).
		Lookup("stock", "groups.list", "_id", "stocks").
		Project(bson.M{"groups": 1, "stocks": listOpt}).Do()).One(&data)

	AddChart(chart, data.Stocks)
	midware.Success(c, data)
}

// 新建分组
func AddGroup(c *gin.Context) {
	req := new(model.Group)
	c.ShouldBind(req)
	uid := c.MustGet("id").(pr.ObjectID)

	if req.Name == "" {
		midware.Error(c, errors.New("分组名为空"))
		return
	}

	g := getGroups(uid)
	for _, i := range g.Groups {
		if i.Name == req.Name {
			midware.Error(c, errors.New("分组名重复"))
			return
		}
	}

	if len(g.Groups) >= 6 {
		midware.Error(c, errors.New("最多创建六个分组"))
		return
	}

	if !req.IsActive {
		req.List = []string{}
	}

	err := db.User.UpdateId(ctx, uid, bson.M{"$addToSet": bson.M{"groups": req}})
	midware.Auto(c, err, "新建分组成功")
}

// 删除分组
func RemGroup(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBind(&req); err != nil {
		midware.Error(c, err)
		return
	}

	uid := c.MustGet("id").(pr.ObjectID)
	g := getGroups(uid)

	for _, gr := range g.Groups {
		if gr.Name == req.Name && gr.IsSys {
			midware.Error(c, errors.New("不能删除该分组"))
			return
		}
	}

	err := db.User.UpdateId(ctx, uid, bson.M{
		"$pull": bson.M{"groups": bson.M{"name": req.Name}},
	})

	midware.Auto(c, err, "已删除分组")
}

// 改变股票分组
func ChangeGroup(c *gin.Context) {
	var req struct {
		Code    string   `json:"code" binding:"required"`
		InGroup []string `json:"inGroup" binding:"required"`
	}
	if err := c.ShouldBind(&req); err != nil {
		midware.Error(c, err)
		return
	}

	uid := c.MustGet("id").(pr.ObjectID)

	group := getGroups(uid)
	bulk := db.User.Bulk()

	for _, g := range group.Groups {
		if g.IsSys || g.IsActive {
			continue
		}
		filter := bson.M{"_id": uid, "groups.name": g.Name}
		// 删除
		bulk.UpdateOne(filter, bson.M{"$pull": bson.M{"groups.$.list": req.Code}})
		if util.In(g.Name, req.InGroup) {
			// 添加至顶部
			bulk.UpdateOne(filter, bson.M{"$push": bson.M{"groups.$.list": bson.M{"$each": bson.A{req.Code}, "$position": 0}}})
		}
	}

	_, err := bulk.Run(ctx)
	midware.Auto(c, err, "修改分组成功")
}

// 股票所在分组
func InGroup(c *gin.Context) {
	code, ok := c.GetQuery("code")
	if !ok {
		midware.Error(c, errors.New("invalid code"))
		return
	}

	uid := c.MustGet("id").(pr.ObjectID)
	groups := getGroups(uid)

	// 获取最近浏览长度
	var length int
	for _, g := range groups.Groups {
		if g.Name == "最近浏览" {
			length = len(g.List)
		}
	}
	bulk := db.User.Bulk()
	filter := bson.M{"_id": uid, "groups.name": "最近浏览"}

	if length >= 50 {
		bulk.UpdateOne(filter, bson.M{"$pop": bson.M{"groups.$.list": 1}})
	}

	// 添加至最近浏览
	bulk.UpdateOne(filter, bson.M{"$pull": bson.M{"groups.$.list": code}}).
		UpdateOne(filter, bson.M{"$push": bson.M{"groups.$.list": bson.M{"$each": bson.A{code}, "$position": 0}}}).
		Run(ctx)

	allGroup := make([]string, 0)
	inGroup := make([]string, 0)

	for _, g := range groups.Groups {
		if g.IsSys || g.IsActive {
			continue
		}
		allGroup = append(allGroup, g.Name)
		if util.In(code, g.List) {
			inGroup = append(inGroup, g.Name)
		}
	}

	midware.Success(c, bson.M{"allGroup": allGroup, "inGroup": inGroup})
}

// 更新动态列表
func PutActiveList(c *gin.Context) {
	req := new(model.Group)
	c.ShouldBind(req)
	uid := c.MustGet("id").(pr.ObjectID)

	req.IsActive = true

	err := db.User.UpdateOne(ctx, bson.M{"_id": uid, "groups.name": req.Name}, bson.M{
		"$set": bson.M{"groups.$": req},
	})

	midware.Auto(c, err, nil)
}
