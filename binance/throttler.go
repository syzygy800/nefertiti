package binance

import (
	"context"
	"time"

	exchange "github.com/adshao/go-binance/v2"
)

var (
	lastRequest       time.Time
	lastWeight        int                                                         = 1
	requestsPerSecond float64                                                     = 0
	BeforeRequest     func(client *Client, method, path string, weight int) error = nil
	AfterRequest      func()                                                      = nil
)

func getIntervalNum(rl exchange.RateLimit) int64 {
	if rl.IntervalNum > 0 {
		return rl.IntervalNum
	}
	return 1
}

func getRequestsPerSecond(info *exchange.ExchangeInfo) int64 {
	for _, rl := range info.RateLimits {
		if rl.RateLimitType == "REQUEST_WEIGHT" {
			if rl.Interval == "SECOND" {
				return rl.Limit / getIntervalNum(rl)
			}
			if rl.Interval == "MINUTE" {
				return (rl.Limit / getIntervalNum(rl)) / 60
			}
			if rl.Interval == "DAY" {
				return (rl.Limit / getIntervalNum(rl)) / (24 * 60 * 60)
			}
		}
	}
	return 20
}

func GetRequestsPerSecond(client *Client, weight int) (float64, error) {
	var out float64 = 20

	if requestsPerSecond == 0 {
		info, err := client.inner.NewExchangeInfoService().Do(context.Background())
		if err != nil {
			client.handleError(err)
			return out, err
		}
		requestsPerSecond = float64(getRequestsPerSecond(info))
	}

	if requestsPerSecond > 0 {
		out = requestsPerSecond
	}

	if lastWeight > 0 {
		out = out / float64(lastWeight)
	}
	lastWeight = weight

	return out, nil
}

func init() {
	BeforeRequest = func(client *Client, method, path string, weight int) error {
		rps, err := GetRequestsPerSecond(client, weight)
		if err != nil {
			return err
		}
		elapsed := time.Since(lastRequest)
		if elapsed.Seconds() < (float64(1) / rps) {
			time.Sleep(time.Duration((float64(time.Second) / rps)) - elapsed)
		}
		return nil
	}
	AfterRequest = func() {
		lastRequest = time.Now()
	}
}
