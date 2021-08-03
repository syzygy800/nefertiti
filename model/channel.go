package model

import (
	"time"
)

type Channel interface {
	Init() error
	GetName() string
	GetOrderType() OrderType
	GetRateLimit() time.Duration
	GetValidity() (time.Duration, error)
	GetMarkets(exchange Exchange, quote Assets, btcVolumeMin float64, valid time.Duration, sandbox, debug bool, ignore []string) (Markets, error)
	GetCalls(exchange Exchange, market string, sandbox, debug bool) (Calls, error)
}
