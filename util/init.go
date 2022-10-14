package util

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Code uint8

const (
	MARKET_CN Code = iota + 1
	MARKET_HK
	MARKET_US

	TYPE_STOCK
	TYPE_INDEX
	TYPE_FUND
	TYPE_IDS
)

func init() {
	zerolog.TimeFieldFormat = "06/01/02 15:04:05"
	zerolog.MessageFieldName = "msg"

	// token
	viper.Set("ts_token", "8dbaa93be7f8d09210ca9cb0843054417e2820203201c0f3f7643410")
	viper.Set("xq_token", "xq_a_token=7b4d94a453e79e9b10174ad2d87da0db78921f7c")

	log.Info().Msg("init config success")
}
