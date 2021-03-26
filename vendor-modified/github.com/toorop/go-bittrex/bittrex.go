// Package Bittrex is an implementation of the Bittrex API in Golang.
package bittrex

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	DEFAULT_HTTP_CLIENT_TIMEOUT = 60
	TIME_FORMAT                 = time.RFC3339
	API_BASE                    = "https://api.bittrex.com"
	API_VERSION                 = "v3"
)

var (
	cooldown           bool
	lastRequest        time.Time
	BeforeRequest      func(path string) (bool, error)      = nil // -> (cooled, error)
	AfterRequest       func()                               = nil
	HandleRateLimitErr func(path string, cooled bool) error = nil
)

const (
	INTENSITY_LOW   = 1  // 1 req/second
	INTENSITY_TWO   = 2  // 0.5 req/second
	INTENSITY_SUPER = 60 // 1 req/minute
)

func RequestsPerSecond(intensity int) float64 {
	return float64(1) / float64(intensity)
}

type Call struct {
	Path      string `json:"path"`
	Intensity int    `json:"intensity"`
}

var Calls = []Call{}

func GetRequestsPerSecond(path string) (float64, bool) { // -> (rps, cooldown)
	if cooldown {
		cooldown = false
		return RequestsPerSecond(INTENSITY_SUPER), true
	}
	for i := range path {
		if strings.Index("?", string(path[i])) > -1 {
			path = path[:i]
			break
		}
	}
	for _, call := range Calls {
		if call.Path == path {
			return RequestsPerSecond(call.Intensity), false
		}
	}
	return RequestsPerSecond(INTENSITY_LOW), false
}

func init() {
	BeforeRequest = func(path string) (bool, error) {
		// accounts are now only to make a maximum of 60 API calls per minute.
		// calls after the limit will fail, with throttle settings automatically resetting at the start of the next minute.
		elapsed := time.Since(lastRequest)
		rps, cooled := GetRequestsPerSecond(path)
		if elapsed.Seconds() < (float64(1) / rps) {
			time.Sleep(time.Duration((float64(time.Second) / rps)) - elapsed)
		}
		return cooled, nil
	}
	AfterRequest = func() {
		lastRequest = time.Now()
	}
	HandleRateLimitErr = func(path string, cooled bool) error {
		var (
			exists bool
		)
		for idx := range path {
			if strings.Index("?", string(path[idx])) > -1 {
				path = path[:idx]
				break
			}
		}
		for idx := range Calls {
			if Calls[idx].Path == path {
				if cooled {
					// rate limited immediately after a cooldown?
					// 1. do another round of "cooling down"
					// 2. do not slow this endpoint down just yet.
				} else {
					Calls[idx].Intensity = Calls[idx].Intensity + 1
				}
				exists = true
			}
		}
		if !exists {
			Calls = append(Calls, Call{
				Path:      path,
				Intensity: INTENSITY_TWO,
			})
		}
		cooldown = true
		return nil
	}
}

func New(apiKey, apiSecret string, appId string) *Bittrex {
	client := NewClient(apiKey, apiSecret)
	return &Bittrex{client, appId}
}

type Bittrex struct {
	client *client
	app_id string
}

func (b *Bittrex) GetMarkets() (markets []Market, err error) {
	var data []byte
	if data, err = b.client.do("GET", "markets", nil, b.app_id, false); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &markets); err != nil {
		return nil, err
	}
	return markets, err
}

func (b *Bittrex) GetTicker(market string) (*Ticker, error) {
	var (
		err  error
		data []byte
	)
	if data, err = b.client.do("GET", fmt.Sprintf("markets/%s/ticker", market), nil, b.app_id, false); err != nil {
		return nil, err
	}
	var out Ticker
	if err = json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *Bittrex) GetMarketSummary(market string) (*MarketSummary, error) {
	var (
		err  error
		data []byte
	)
	if data, err = b.client.do("GET", fmt.Sprintf("markets/%s/summary", market), nil, b.app_id, false); err != nil {
		return nil, err
	}
	var out MarketSummary
	if err = json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *Bittrex) GetOrderBook(market string, depth int) (*OrderBook, error) {
	if depth > 500 {
		depth = 500
	} else if depth < 1 {
		depth = 1
	}
	var (
		err  error
		data []byte
	)
	if data, err = b.client.do("GET", fmt.Sprintf("markets/%s/orderbook?depth=%d", market, depth), nil, b.app_id, false); err != nil {
		return nil, err
	}
	var out OrderBook
	if err = json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *Bittrex) CreateOrder(
	marketSymbol string,
	direction string,
	orderType string,
	quantity float64,
	limit float64,
	timeInForce string,
) (*Order, error) {
	var err error

	type newOrder struct {
		MarketSymbol string `json:"marketSymbol"`
		Direction    string `json:"direction"`
		OrderType    string `json:"type"`
		Quantity     string `json:"quantity"`
		Limit        string `json:"limit,omitempty"`
		TimeInForce  string `json:"timeInForce,omitempty"`
	}

	var order *newOrder
	order = &newOrder{
		MarketSymbol: marketSymbol,
		Direction:    direction,
		OrderType:    orderType,
		Quantity:     strconv.FormatFloat(quantity, 'f', -1, 64),
		TimeInForce:  timeInForce,
	}

	if limit > 0 {
		order.Limit = strconv.FormatFloat(limit, 'f', -1, 64)
	}

	var payload []byte
	if payload, err = json.Marshal(order); err != nil {
		return nil, err
	}

	var data []byte
	if data, err = b.client.do("POST", "orders", payload, b.app_id, true); err != nil {
		return nil, err
	}

	var result Order
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (b *Bittrex) CancelOrder(orderId string) (err error) {
	_, err = b.client.do("DELETE", fmt.Sprintf("orders/%s", orderId), nil, b.app_id, true)
	return err
}

func (b *Bittrex) GetOpenOrders(market string) (orders Orders, err error) {
	path := func() string {
		result := "orders/open"
		if market != "" && market != "all" {
			result += "?marketSymbol=" + market

		}
		return result
	}
	var data []byte
	if data, err = b.client.do("GET", path(), nil, b.app_id, true); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func (b *Bittrex) GetOrder(orderId string) (*Order, error) {
	var (
		err  error
		data []byte
	)
	if data, err = b.client.do("GET", fmt.Sprintf("orders/%s", orderId), nil, b.app_id, true); err != nil {
		return nil, err
	}
	var order Order
	if err = json.Unmarshal(data, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (b *Bittrex) GetOrderHistory(market string) (Orders, error) {
	path := func(nextPageToken string) string {
		result := "orders/closed?pageSize=200"
		if market != "" && market != "all" {
			result += "&marketSymbol=" + market
		}
		if nextPageToken != "" {
			result += "&nextPageToken=" + nextPageToken
		}
		return result
	}

	get := func(nextPageToken string) (Orders, error) {
		var (
			err    error
			data   []byte
			result Orders
		)
		if data, err = b.client.do("GET", path(nextPageToken), nil, b.app_id, true); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	var (
		err    error
		page   Orders
		result Orders
	)
	if page, err = get(""); err != nil {
		return nil, err
	}
	for len(page) > 0 {
		result = append(result, page...)
		if page, err = get(page[len(page)-1].Id); err != nil {
			return nil, err
		}
	}

	return result, nil
}
