package midware

import (
	"context"
	"errors"
	"fund/db"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var limit = rate.NewLimiter(20, 1000)

// 流量控制
func FlowController(c *gin.Context) {
	ip := c.ClientIP()
	if ip == "127.0.0.1" {
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	if err := limit.Wait(c); err != nil {
		Error(c, errors.New("服务器繁忙"))
		return
	}

	ok, _ := db.LimitDB.Exists(ctx, ip).Result()
	if ok == 1 {
		times, _ := db.LimitDB.Incr(ctx, ip).Result()
		if times > 250 {
			Error(c, errors.New("请不要频繁访问接口"), http.StatusForbidden)
		}
	} else {
		db.LimitDB.SetEX(ctx, ip, 1, time.Minute)
	}
}
