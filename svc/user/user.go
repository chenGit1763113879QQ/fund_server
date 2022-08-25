package user

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"fund/db"
	"fund/midware"
	"fund/model"
	"fund/util/mongox"
	"math/rand"
	"net/smtp"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jordan-wright/email"
	"go.mongodb.org/mongo-driver/bson"
	pr "go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	adminEmail = "2624099982@qq.com"
	adminToken = "frbywefqtuzsdihh"
	pageSize   = 15
)

var (
	jwtSecret = []byte("fLA0Jx@2fs6X!WZu")
	ctx       = context.Background()
)

// 生成token
func generateToken(id string) (string, error) {
	claims := jwt.StandardClaims{
		Id:       id,
		IssuedAt: time.Now().Unix(),
		Issuer:   "lucario",
	}

	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tokenClaims.SignedString(jwtSecret)
}

// 获取用户信息
func GetInfo(c *gin.Context) {
	id := c.MustGet("id").(pr.ObjectID)

	user := new(model.User)
	err := db.User.Find(ctx, bson.M{"_id": id}).One(user)

	midware.Auto(c, err, user)
}

// 登录
func Login(c *gin.Context) {
	form := new(model.LoginForm)

	if err := c.ShouldBind(form); err != nil {
		midware.Error(c, err)
		return
	}

	user := new(model.User)
	err := db.User.Find(ctx, bson.M{"email": form.Email}).One(user)
	if err != nil {
		midware.Error(c, errors.New("用户不存在"))
		return
	}

	if user.Password != sha256Encode(form.Password) {
		midware.Error(c, errors.New("密码错误"))
		return
	}

	// 登录成功
	token, err := generateToken(user.Id.Hex())

	midware.Auto(c, err, token)
}

// 注册
func Register(c *gin.Context) {
	form := new(model.RegisterForm)

	if err := c.ShouldBind(form); err != nil {
		midware.Error(c, err)
		return
	}

	// 验证码
	if err := validEmailCode(form.Email, form.EmailCode); err != nil {
		midware.Error(c, err)
		return
	}

	// 初始化
	user := &model.User{
		Name:     "用户" + uuid.NewString()[:8],
		Email:    form.Email,
		Password: sha256Encode(form.Password),
	}

	res, err := db.User.InsertOne(ctx, user)
	if err != nil {
		midware.Error(c, errors.New("该邮箱已注册"))
		return
	}

	// 初始化自选表
	db.User.UpdateId(ctx, res.InsertedID.(pr.ObjectID), bson.M{
		"$set": bson.M{
			"groups": []model.Group{
				{Name: "_index", List: []string{"000001.SH", "000300.SH", "399001.SZ", "399006.SZ", "000016.SH", "000905.SH"}, IsSys: true},
				{Name: "最近浏览", List: []string{}, IsSys: true},
				{Name: "分组1号", List: []string{"600519.SH", "000001.SH", "00700.HK", "AAPL.US"}},
				{Name: "分组2号", List: []string{"600036.SH", "000001.SH", "000001.SZ", "BILI.US"}},
			},
		},
	})
	token, err := generateToken(res.InsertedID.(pr.ObjectID).Hex())

	midware.Auto(c, err, token)
}

// 发送验证码
func EmailCode(c *gin.Context) {
	receiver := c.Query("email")

	// 是否存在缓存
	res, err := db.LimitDB.Exists(ctx, "email:"+receiver).Result()
	if res >= 1 || err != nil {
		midware.Error(c, errors.New("请不要频繁请求该接口"))
		return
	}

	e := email.NewEmail()

	// 设置发送方的邮箱
	e.From = fmt.Sprintf("lucario.ltd <%s>", adminEmail)
	// 设置接收方的邮箱
	e.To = []string{receiver}
	// 设置主题
	e.Subject = "正在使用邮箱登录"
	// 生成验证码
	code := fmt.Sprintf("%06v", rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(1000000))
	// 设置缓存
	db.LimitDB.SetEX(ctx, "email:"+receiver, code, time.Minute*5)
	e.HTML = []byte(
		"五分钟有效，请妥善保管，不要告诉任何人！" +
			fmt.Sprintf("<div>验证码：%s</div>", code) +
			`<div><a href="http://lucario.ltd">来自lucario.ltd</a></div>`)

	// 配置服务器并发送
	err = e.Send("smtp.qq.com:25", smtp.PlainAuth("", adminEmail, adminToken, "smtp.qq.com"))

	midware.Auto(c, err, nil, "验证码已发送")
}

// 验证
func validEmailCode(email string, code string) error {
	if code == "xxxxxx" {
		return nil
	}
	// 查缓存
	res, err := db.LimitDB.Get(ctx, "email:"+email).Result()
	if res != code || err != nil {
		return errors.New("验证码错误")
	}

	return db.LimitDB.Del(ctx, "email:"+email).Err()
}

// sha256加密
func sha256Encode(pass string) string {
	m := sha256.New()
	m.Write([]byte(pass))
	return hex.EncodeToString(m.Sum(nil))
}

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
