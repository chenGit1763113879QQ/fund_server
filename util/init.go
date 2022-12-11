package util

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Code string

const (
	// market
	CN = "CN"
	HK = "HK"
	US = "US"

	// type
	STOCK = "stock"
	INDEX = "index"
	FUND  = "fund"
	IDS   = "ids"
)

func init() {
	zerolog.TimeFieldFormat = "06/01/02 15:04:05"
	zerolog.MessageFieldName = "msg"

	// token
	viper.Set("ts_token", "8dbaa93be7f8d09210ca9cb0843054417e2820203201c0f3f7643410")
	viper.Set("xq_token", "bid=3c6bb14598fe9ac45474be34ecb46d45_l2zr5ald; xq_is_login=1; u=3611404155; device_id=ac1de765a9ff0714bc831b960b33702a; xq_a_token=eca6e34c969d3d0b5b4649c099cce429221332dd; xqat=eca6e34c969d3d0b5b4649c099cce429221332dd; xq_id_token=eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.eyJ1aWQiOjM2MTE0MDQxNTUsImlzcyI6InVjIiwiZXhwIjoxNjcyMTQzMTE0LCJjdG0iOjE2Njk1NTExMTQyNDAsImNpZCI6ImQ5ZDBuNEFadXAifQ.BUuE_DCJlKTFJGZ8Q2O8AM04B9nBYYn8xwCd9YotVaBjHNCmp2pwxNk_QDryuNjNDvJ39FTHkFq-Kz1cIdCHMjiFXCyRJVexLY7VjGK0CTXq0TfDvPYJF-Lw_94gA4kRngMldMkGvr81U_QabrjkUhWkXMGo2Lw6knK42KoWXKLZI4pRfT4BEhTWoksS_R41kfmFPx2cuojgMpHBxPNllf1Xj53ykel1W5LK294HU4BEO95nPEoKFpKRhnH1sgJdEqYr1WFa5n3HcE1Vp2Zha56ORM7_Z5BJaFvbfFhViipE1OfC0GHjYR2Hu51HfwWjyDKP5qNtNm2amE43Tc6lyA; xq_r_token=6c69ae0a10a10eb6a095f5421ec2ae3fb25fd361; s=bs11gxqrz7; acw_tc=2760779c16707665521593730e46efcb3f1e567ceeb52e115bd90925931da3")

	log.Info().Msg("init config success")
}
