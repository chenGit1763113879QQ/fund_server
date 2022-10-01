package model

import (
	"github.com/qiniu/qmgo/field"
)

type User struct {
	field.DefaultField `bson:",inline"`
	Email              string `json:"email"`
	Name               string `json:"name"`
	Password           string `json:"-"`
}

type RegisterForm struct {
	Email     string `json:"email" binding:"email"`
	Password  string `json:"password" binding:"min=6,max=16"`
	EmailCode string `json:"email_code" binding:"len=6"`
}

type LoginForm struct {
	Email    string `json:"email" binding:"email"`
	Password string `json:"password" binding:"min=6,max=16"`
}
