package midware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	// fast encode
	encoder, _ = zstd.NewWriter(nil, zstd.WithEncoderLevel(1))
)

func Auto(c *gin.Context, err error, data any, msg ...string) {
	if err != nil {
		Warning(c, err.Error())
	} else {
		if len(msg) > 0 {
			Success(c, data, msg[0])
		} else {
			Success(c, data)
		}
	}
}

func Error(c *gin.Context, err error, code ...int) {
	if len(code) > 0 {
		c.AbortWithStatusJSON(code[0], bson.M{
			"status": false, "msg": err.Error(),
		})
	} else {
		c.AbortWithStatusJSON(http.StatusBadRequest, bson.M{
			"status": false, "msg": err.Error(),
		})
	}
}

func Success(c *gin.Context, data any, msg ...string) {
	if len(msg) > 0 {
		c.AbortWithStatusJSON(http.StatusOK, bson.M{
			"status": true, "data": data, "msg": msg[0],
		})
	} else {
		c.AbortWithStatusJSON(http.StatusOK, bson.M{
			"status": true, "data": data,
		})
	}
}

func Warning(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusOK, bson.M{
		"status": false, "msg": msg,
	})
}

func Zip(data any) []byte {
	src, _ := json.Marshal(&data)
	return encoder.EncodeAll(src, nil)
}
