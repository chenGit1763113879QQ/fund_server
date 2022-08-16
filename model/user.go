package model

import (
	"github.com/qiniu/qmgo/field"
	pr "go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	field.DefaultField `bson:",inline"`
	Email              string     `json:"email"`
	Name               string     `json:"name"`
	Password           string     `json:"-"`
	Description        string     `json:"description"`
	Points             int        `json:"points"`
	Ip                 IpLocation `json:"ip"`
}

type IpLocation struct {
	Country string `json:"country"`
	Region  string `json:"regionName"`
	City    string `json:"city"`
	Ip      string `json:"-"`
}

type RegisterForm struct {
	Email     string `json:"email" binding:"email"`
	Password  string `json:"password" binding:"min=6,max=16"`
	EmailCode string `json:"email_code" binding:"len=6"`
	IP        string
}

type LoginForm struct {
	Email    string `json:"email" binding:"email"`
	Password string `json:"password" binding:"min=6,max=16"`
	IP       string `binding:"ip"`
}

type UpdateForm struct {
	Id          pr.ObjectID `bson:"-"`
	Name        string      `json:"name" binding:"min=2,max=12"`
	Password    string      `bson:"password,omitempty" json:"password" binding:"min=6,max=16"`
	Description string      `json:"description" binding:"max=32"`
}
