package main

import (
	"errors"
	"fund/midware"
	"fund/svc/stock"
	"fund/svc/user"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

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
		GET("/predict", stock.PredictKline).
		GET("/portfolio", stock.GetPortfolio).
		GET("/hot", stock.GetHotStock)

	api.Group("back").
		GET("/logs", stock.GetBackLogs)

	api.Group("/market").
		GET("/bk", stock.AllBKDetails)

	r.NoRoute(func(c *gin.Context) {
		// api not found
		if c.Request.URL.Path == "/api" || strings.HasPrefix(c.Request.URL.Path, "/api/") {
			midware.Error(c, errors.New("page not found"), http.StatusNotFound)
			return
		}
		// static fallback
		c.File("public/index.html")
	})

	panic(r.Run("0.0.0.0:10888"))
}
