package midware

import (
	"errors"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var jwtSecret = []byte("fLA0Jx@2fs6X!WZu")

func Authorize(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		token = c.Query("Authorization")
	}

	claims, err := PhaseToken(token)
	if err != nil {
		Error(c, err, http.StatusUnauthorized)
		return
	}

	id, err := primitive.ObjectIDFromHex(claims.Id)
	if id.IsZero() {
		Error(c, err, http.StatusUnauthorized)
		return
	}

	c.Set("id", id)
}

// 解析token
func PhaseToken(token string) (*jwt.StandardClaims, error) {
	claims := new(jwt.StandardClaims)

	tokenClaims, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	// 解密
	claims, ok := tokenClaims.Claims.(*jwt.StandardClaims)
	if !ok {
		return nil, errors.New("parse error")
	}
	return claims, nil
}
