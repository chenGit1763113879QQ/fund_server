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
	viper.Set("xq_token", "xq_a_token=f273f18b14b700bba7ab3febf7510da0346bdb9f")

	log.Info().Msg("init config success")
}
