package util

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
