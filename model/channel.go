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
	GetMarkets(exchange Exchange, quote string, btc_volume_min, btc_pump_max float64, valid time.Duration, sandbox, debug bool) (Markets, error)
	GetCalls(exchange Exchange, market string, sandbox, debug bool) (Calls, error)
}
