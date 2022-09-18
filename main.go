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
	zerolog.TimeFieldFormat = "06/01/02 15:04:05"
	zerolog.MessageFieldName = "msg"
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(midware.FlowController)

	api := r.Group("/api")
	ws := r.Group("/ws")

	api.Group("/user").
		GET("/emailCode", user.EmailCode).
		POST("/login", user.Login).
		POST("/newAuth", user.Register)

	// api.Use(midware.Authorize)
	// ws.Use(midware.Authorize)

	ws.Group("/stock").
		GET("/detail", stock.ConnectItems).
		GET("/market", stock.ConnectMarket)

	ws.GET("/notify", stock.Notify)

	api.Group("/stock").
		GET("/search", stock.Search).
		GET("/list", stock.GetStockList).
		GET("/chart/kline", stock.GetKline).
		GET("/predict", stock.PredictKline).
		GET("/group", stock.GetGroups).
		POST("/group", stock.AddGroup).
		PUT("/group", stock.ChangeGroup).
		DELETE("/group", stock.RemGroup).
		GET("/group/in", stock.InGroup)

	api.Group("back").
		GET("/logs", stock.GetBackLogs)

	api.Group("/market").
		GET("/bk", stock.AllBKDetails)

	api.Group("/article").
		GET("/:id", user.GetArticle).
		GET("/list/news", user.GetNews)

	r.NoRoute(func(c *gin.Context) {
		midware.Error(c, errors.New("page not found"), http.StatusNotFound)
	})

	panic(r.Run("127.0.0.1:10888"))
}
