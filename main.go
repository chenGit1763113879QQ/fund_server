package main

import (
	"errors"
	"fund/midware"
	"fund/svc/pro"
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

	pro.Test()

	r.Use(midware.FlowController)

	api := r.Group("/api")
	ws := r.Group("/ws")

	api.Group("/user").
		GET("/emailCode", user.EmailCode).
		POST("/login", user.Login).
		POST("/newAuth", user.Register).
		GET("/info", midware.Authorize, user.GetInfo)

	api.Use(midware.Authorize)
	ws.Use(midware.Authorize)

	ws.Group("/stock").
		GET("/list", stock.ConnectCList).
		GET("/detail", stock.ConnectItems).
		GET("/market", stock.ConnectMarket)

	api.Group("/stock").
		GET("/search", stock.Search).
		GET("/all", stock.GetAllStock).
		GET("/list", stock.GetStockList).
		GET("/chart/kline", stock.GetKline).
		GET("/predict", stock.PredictKline).
		GET("/center/:path", stock.DataCenter).
		GET("/group", stock.GetGroups).
		POST("/group", stock.AddGroup).
		PUT("/group", stock.ChangeGroup).
		DELETE("/group", stock.RemGroup).
		POST("/active", stock.GetActiveList).
		PUT("/active", stock.PutActiveList).
		GET("/group/in", stock.InGroup)

	api.Group("/market").
		GET("/bk", stock.DetailBK)

	api.Group("/article").
		GET("/:id", user.GetArticle).
		GET("/list/news", user.GetNews)

	r.NoRoute(func(c *gin.Context) {
		midware.Error(c, errors.New("page not found"), http.StatusNotFound)
	})

	panic(r.Run("127.0.0.1:10888"))
}
