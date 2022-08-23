package main

import (
	"errors"
	"fund/midware"
	"fund/svc/stock"
	"fund/svc/user"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func init() {
	zerolog.TimeFieldFormat = "2006/01/02 15:04:05"
	zerolog.MessageFieldName = "msg"
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(midware.FlowController)

	api := r.Group("/api")
	ws := r.Group("/ws")

	User := api.Group("/user")
	User.GET("/emailCode", user.EmailCode)
	User.POST("/login", user.Login)
	User.POST("/newAuth", user.Register)

	User.GET("/:id", user.GetArticle)
	User.GET("/list/news", user.GetNews)

	User.Use(midware.Authorize)
	User.GET("/info", user.GetInfo)

	api.Use(midware.Authorize)
	ws.Use(midware.Authorize)

	wsStock := ws.Group("/stock")
	wsStock.GET("/list", stock.ConnectCList)
	wsStock.GET("/detail", stock.ConnectItems)
	wsStock.GET("/market", stock.ConnectMarket)

	stk := api.Group("/stock")
	stk.GET("/search", stock.Search)
	stk.GET("/all", stock.GetAllStock)
	stk.GET("/list", stock.GetStockList)
	stk.GET("/chart/kline", stock.GetKline)
	stk.GET("/predict", stock.PredictKline)
	stk.GET("/center/:path", stock.DataCenter)

	stk.GET("/group", stock.GetGroups)
	stk.POST("/group", stock.AddGroup)
	stk.PUT("/group", stock.ChangeGroup)
	stk.DELETE("/group", stock.RemGroup)

	stk.POST("/active", stock.GetActiveList)
	stk.PUT("/active", stock.PutActiveList)
	stk.GET("/group/in", stock.InGroup)

	market := api.Group("/market")
	market.GET("/bk", stock.DetailBK)

	r.NoRoute(func(c *gin.Context) {
		midware.Error(c, errors.New("page not found"), http.StatusNotFound)
	})

	panic(r.Run("127.0.0.1:10888"))
}
