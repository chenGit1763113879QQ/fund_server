package midware

import (
	"context"
	"errors"
	"fund/db"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	poolSize = 2000
)

var ctx = context.Background()

func init() {
	go func() {
		for {
			length, _ := db.LimitDB.LLen(ctx, "global").Result()
			if length < poolSize {
				db.LimitDB.LPush(ctx, "global", 1)
			}
			time.Sleep(poolSize / time.Minute)
		}
	}()
}

// 流量控制
func FlowController(c *gin.Context) {
	ip := c.ClientIP()

	r, err := db.LimitDB.LPop(ctx, "global").Result()
	if r == "" || err != nil {
		Error(c, errors.New("server is busy"), http.StatusForbidden)
		return
	}

	ok, _ := db.LimitDB.Exists(ctx, ip).Result()
	if ok == 1 {
		times, _ := db.LimitDB.Incr(ctx, ip).Result()
		if times > 250 {
			Error(c, errors.New("api forbidden"), http.StatusForbidden)
		}
	} else {
		db.LimitDB.SetEX(ctx, ip, 1, time.Minute)
	}
}
