package main

import (
	"time"

	"github.com/spf13/viper"
)

func init() {
	// token
	viper.SetDefault("ts_token", "8dbaa93be7f8d09210ca9cb0843054417e2820203201c0f3f7643410")
	viper.SetDefault("xq_token", "xq_a_token=80b283f898285a9e82e2e80cf77e5a4051435344")

	// kline
	viper.SetDefault("kline.update.duration", time.Hour*12)
}
