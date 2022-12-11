package main

import (
	"fund/svc/stock"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	api := r.Group("/api")
	ws := r.Group("/ws")

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

	api.Group("/market").
		GET("/bk", stock.AllBKDetails)

	panic(r.Run("0.0.0.0:10888"))
}
