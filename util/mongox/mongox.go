/*
Name: mongox
Auth: lucario
Date: 2022/08/12
Desc: 基于 mongo 的 pipeline 链式调用封装
*/
package mongox

import "go.mongodb.org/mongo-driver/bson"

type options struct {
	opts []bson.D
}

func Pipeline() *options {
	return &options{opts: make([]bson.D, 0)}
}

func (p *options) Lookup(from, local, foreign, as string) *options {
	p.opts = append(p.opts, bson.D{{Key: "$lookup", Value: bson.M{
		"from":         from,
		"localField":   local,
		"foreignField": foreign,
		"as":           as,
	}}})
	return p
}

func (p *options) Match(opt bson.M) *options {
	p.opts = append(p.opts, bson.D{{Key: "$match", Value: opt}})
	return p
}

func (p *options) Unwind(path string) *options {
	p.opts = append(p.opts, bson.D{{Key: "$unwind", Value: bson.M{"path": path}}})
	return p
}

func (p *options) Project(opt bson.M) *options {
	p.opts = append(p.opts, bson.D{{Key: "$project", Value: opt}})
	return p
}

func (p *options) Sort(opt bson.M) *options {
	p.opts = append(p.opts, bson.D{{Key: "$sort", Value: opt}})
	return p
}

func (p *options) Group(opt bson.M) *options {
	p.opts = append(p.opts, bson.D{{Key: "$group", Value: opt}})
	return p
}

func (p *options) Limit(limit int) *options {
	p.opts = append(p.opts, bson.D{{Key: "$limit", Value: limit}})
	return p
}

func (p *options) Skip(skip int) *options {
	p.opts = append(p.opts, bson.D{{Key: "$skip", Value: skip}})
	return p
}

func (p *options) Bucket(groupBy string, boundaries bson.A, defaults string, output bson.M) *options {
	p.opts = append(p.opts, bson.D{{Key: "$bucket", Value: bson.M{
		"groupBy":    groupBy,
		"boundaries": boundaries,
		"default":    defaults,
		"output":     output,
	}}})
	return p
}

func (p *options) Do() []bson.D {
	return p.opts
}
