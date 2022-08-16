package model

import (
	"time"

	"github.com/qiniu/qmgo/field"
	pr "go.mongodb.org/mongo-driver/bson/primitive"
)

type ArtOpt struct {
	Uid  pr.ObjectID
	Code string `form:"code"`
	Sort string `form:"sort"`
	Type string `form:"type" binding:"oneof=0 1 2"`
	Page int  `form:"page"`
}

type Article struct {
	field.DefaultField `bson:",inline"`
	Uid                pr.ObjectID
	Title              string        `json:"title" binding:"max=32"`
	Content            string        `json:"content" binding:"max=1024,required"`
	CreateAt           time.Time     `bson:"createAt"`
	Reads              []pr.ObjectID `json:"reads"`
	Likes              []pr.ObjectID `json:"likes"`
	Colls              []pr.ObjectID `json:"colls"`
	Comments           int           `json:"comments"`
	Type               int           `json:"type" binding:"oneof=0 1 2"` // 0文章 1讨论 2资讯
	Tag                []string      `json:"tag" binding:"max=5"`
}

type Comment struct {
	Id       string `bson:"-" json:"id"`
	Content  string `json:"content" binding:"max=256,required"`
	Uid      pr.ObjectID
	Aid      pr.ObjectID
	ParentId pr.ObjectID `bson:",omitempty"`
	Likes    int         `bson:",omitempty" json:"likes"`
	Replies  int         `bson:",omitempty" json:"replies"`
	CreateAt time.Time   `bson:"createAt"`
}
