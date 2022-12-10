package util

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Code uint8

const (
	// market
	CN Code = iota + 1
	HK
	US

	// type
	STOCK
	INDEX
	FUND
	IDS
)

func init() {
	zerolog.TimeFieldFormat = "06/01/02 15:04:05"
	zerolog.MessageFieldName = "msg"

	// token
	viper.Set("ts_token", "8dbaa93be7f8d09210ca9cb0843054417e2820203201c0f3f7643410")
	viper.Set("xq_token", "xq_a_token=eca6e34c969d3d0b5b4649c099cce429221332dd")

	log.Info().Msg("init config success")
}
