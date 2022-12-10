package main

import (
	"fund/midware"
	"fund/svc/stock"
	"fund/svc/user"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	api := r.Group("/api")
	ws := r.Group("/ws")

	api.Group("/user").
		GET("/emailCode", user.EmailCode).
		POST("/login", user.Login).
		POST("/newAuth", user.Register)

	api.Use(midware.Authorize)
	ws.Use(midware.Authorize)

	ws.Group("/stock").
		GET("/detail", stock.ConnectItems).
		GET("/market", stock.ConnectMarket)

	ws.GET("/notify", stock.Notify)

	api.Group("/stock").
		GET("/search", stock.Search).
		GET("/list", stock.GetStockList).
		GET("/chart/kline", stock.GetKline).
		GET("/portfolio", stock.GetPortfolio).
		GET("/hot", stock.GetHotStock)

	api.Group("back").
		GET("/logs", stock.GetBackLogs)

	api.Group("/market").
		GET("/bk", stock.AllBKDetails)

	panic(r.Run("0.0.0.0:10888"))
}
