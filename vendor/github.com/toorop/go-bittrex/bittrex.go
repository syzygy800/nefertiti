// Package Bittrex is an implementation of the Bittrex API in Golang.
package bittrex

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	DEFAULT_HTTP_CLIENT_TIMEOUT = 60
	TIME_FORMAT                 = "2006-01-02T15:04:05"
	TIME_FORMAT_V3              = time.RFC3339
)

// Bittrex API endpoint
func fmtApiBase(version int) string {
	switch version {
	case 2:
		return "https://international.bittrex.com/api"
	case 3:
		return "https://api.bittrex.com"
	default:
		return "https://api.bittrex.com/api"
	}
}

// Bittrex API version
func fmtApiVersion(version int) string {
	switch version {
	case 2:
		return "v2.0"
	case 3:
		return "v3"
	default:
		return "v1.1"
	}
}

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

// New1 returns a v1 Bittrex struct
func New1(apiKey, apiSecret string) *Bittrex1 {
	client := NewClient(apiKey, apiSecret)
	return &Bittrex1{client}
}

// handleErr gets JSON response from Bittrex v1 API en deal with error
func handleErr(path string, resp Response) error {
	if !resp.Success {
		if strings.Contains(resp.Message, "was throttled") {
			err := HandleRateLimitErr(path, false)
			if err != nil {
				return err
			}
		}
		return errors.New(resp.Message)
	}
	return nil
}

// Bittrex1 represent a v1 Bittrex client
type Bittrex1 struct {
	client *client
}

// GetCurrencies is used to get all supported currencies at Bittrex along with other meta data.
func (b *Bittrex1) GetCurrencies() (currencies []Currency1, err error) {
	var (
		data []byte
		resp Response
	)
	const (
		path = "public/getcurrencies"
	)
	data, err = b.client.do("GET", 1, path, "", false)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	err = json.Unmarshal(resp.Result, &currencies)
	return
}

// GetMarkets is used to get the open and available trading markets at Bittrex along with other meta data.
func (b *Bittrex1) GetMarkets() (markets []Market1, err error) {
	var (
		data []byte
		resp Response
	)
	const (
		path = "public/getmarkets"
	)
	data, err = b.client.do("GET", 1, path, "", false)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	err = json.Unmarshal(resp.Result, &markets)
	return
}

// GetTicker is used to get the current ticker values for a market.
func (b *Bittrex1) GetTicker(market string) (ticker Ticker1, err error) {
	var (
		data []byte
		resp Response
	)
	const (
		path = "public/getticker"
	)
	data, err = b.client.do("GET", 1, fmt.Sprintf("%s?market=%s", path, strings.ToUpper(market)), "", false)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	err = json.Unmarshal(resp.Result, &ticker)
	return
}

// GetMarketSummaries is used to get the last 24 hour summary of all active exchanges
func (b *Bittrex1) GetMarketSummaries() (marketSummaries []MarketSummary1, err error) {
	var (
		data []byte
		resp Response
	)
	const (
		path = "public/getmarketsummaries"
	)
	data, err = b.client.do("GET", 1, path, "", false)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	err = json.Unmarshal(resp.Result, &marketSummaries)
	return
}

// GetMarketSummary is used to get the last 24 hour summary for a given market
func (b *Bittrex1) GetMarketSummary(market string) (marketSummary []MarketSummary1, err error) {
	var (
		data []byte
		resp Response
	)
	const (
		path = "public/getmarketsummary"
	)
	data, err = b.client.do("GET", 1, fmt.Sprintf("%s?market=%s", path, strings.ToUpper(market)), "", false)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	err = json.Unmarshal(resp.Result, &marketSummary)
	return
}

// GetOrderBook is used to get retrieve the orderbook for a given market
// market: a string literal for the market (ex: BTC-LTC)
// cat: buy, sell or both to identify the type of orderbook to return.
// depth: how deep of an order book to retrieve. Max is 100
func (b *Bittrex1) GetOrderBook(market, cat string, depth int) (orderBook OrderBook1, err error) {
	var (
		data []byte
		resp Response
	)
	const (
		path = "public/getorderbook"
	)
	if cat != "buy" && cat != "sell" && cat != "both" {
		cat = "both"
	}
	if depth > 100 {
		depth = 100
	}
	if depth < 1 {
		depth = 1
	}
	data, err = b.client.do("GET", 1, fmt.Sprintf("%s?market=%s&type=%s&depth=%d", path, strings.ToUpper(market), cat, depth), "", false)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	if cat == "buy" {
		err = json.Unmarshal(resp.Result, &orderBook.Buy)
	} else if cat == "sell" {
		err = json.Unmarshal(resp.Result, &orderBook.Sell)
	} else {
		err = json.Unmarshal(resp.Result, &orderBook)
	}
	return
}

// GetOrderBookBuySell is used to get retrieve the buy or sell side of an orderbook for a given market
// market: a string literal for the market (ex: BTC-LTC)
// cat: buy or sell to identify the type of orderbook to return.
// depth: how deep of an order book to retrieve. Max is 100
func (b *Bittrex1) GetOrderBookBuySell(market, cat string, depth int) (orderBook []BookEntry1, err error) {
	var (
		data []byte
		resp Response
	)
	const (
		path = "public/getorderbook"
	)
	if cat != "buy" && cat != "sell" {
		cat = "buy"
	}
	if depth > 100 {
		depth = 100
	}
	if depth < 1 {
		depth = 1
	}
	data, err = b.client.do("GET", 1,
		fmt.Sprintf("%s?market=%s&type=%s&depth=%d",
			path,
			strings.ToUpper(market),
			cat,
			depth,
		), "", false)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	err = json.Unmarshal(resp.Result, &orderBook)
	return
}

// BuyLimit is used to place a limited buy order in a specific market.
func (b *Bittrex1) BuyLimit(market string, quantity, rate float64) (uuid string, err error) {
	var (
		data []byte
		resp Response
	)
	const (
		path = "market/buylimit"
	)
	data, err = b.client.do("GET", 1,
		fmt.Sprintf("%s?market=%s&quantity=%s&rate=%s",
			path,
			market,
			strconv.FormatFloat(quantity, 'f', 8, 64),
			strconv.FormatFloat(rate, 'f', 8, 64),
		), "", true)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	var out UUID
	if err = json.Unmarshal(resp.Result, &out); err != nil {
		return
	}
	uuid = out.UUID
	return
}

// BuyMarket is used to place a market buy order in a spacific market.
func (b *Bittrex1) BuyMarket(market string, quantity float64) (uuid string, err error) {
	var (
		data []byte
		resp Response
	)
	const (
		path = "market/buymarket"
	)
	data, err = b.client.do("GET", 1,
		fmt.Sprintf("%s?market=%s&quantity=%s",
			path,
			market,
			strconv.FormatFloat(quantity, 'f', 8, 64),
		), "", true)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	var out UUID
	if err = json.Unmarshal(resp.Result, &out); err != nil {
		return
	}
	uuid = out.UUID
	return
}

// SellLimit is used to place a limited sell order in a specific market.
func (b *Bittrex1) SellLimit(market string, quantity, rate float64) (uuid string, err error) {
	var (
		data []byte
		resp Response
	)
	const (
		path = "market/selllimit"
	)
	data, err = b.client.do("GET", 1,
		fmt.Sprintf("%s?market=%s&quantity=%s&rate=%s",
			path,
			market,
			strconv.FormatFloat(quantity, 'f', 8, 64),
			strconv.FormatFloat(rate, 'f', 8, 64),
		), "", true)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	var out UUID
	if err = json.Unmarshal(resp.Result, &out); err != nil {
		return
	}
	uuid = out.UUID
	return
}

// SellMarket is used to place a market sell order in a specific market.
func (b *Bittrex1) SellMarket(market string, quantity float64) (uuid string, err error) {
	var (
		data []byte
		resp Response
	)
	const (
		path = "market/sellmarket"
	)
	data, err = b.client.do("GET", 1,
		fmt.Sprintf("%s?market=%s&quantity=%s",
			path,
			market,
			strconv.FormatFloat(quantity, 'f', 8, 64),
		), "", true)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	var out UUID
	if err = json.Unmarshal(resp.Result, &out); err != nil {
		return
	}
	uuid = out.UUID
	return
}

// CancelOrder is used to cancel a buy or sell order.
func (b *Bittrex1) CancelOrder(orderID string) (err error) {
	var (
		data []byte
		resp Response
	)
	const (
		path = "market/cancel"
	)
	data, err = b.client.do("GET", 1, fmt.Sprintf("%s?uuid=%s", path, orderID), "", true)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	err = handleErr(path, resp)
	return
}

// GetOpenOrders returns orders that you currently have opened.
// If market is set to "all", GetOpenOrders return all orders
// If market is set to a specific order, GetOpenOrders return orders for this market
func (b *Bittrex1) GetOpenOrders(market string) (orders Orders, err error) {
	var (
		data []byte
		resp Response
	)
	path := "market/getopenorders"
	if market != "all" {
		path += "?market=" + strings.ToUpper(market)
	}
	data, err = b.client.do("GET", 1, path, "", true)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	err = json.Unmarshal(resp.Result, &orders)
	return
}

// GetOrderHistory used to retrieve your order history.
// market string literal for the market (ie. BTC-LTC). If set to "all", will return for all market
func (b *Bittrex1) GetOrderHistory(market string) (fills Fills, err error) {
	var (
		data []byte
		resp Response
	)
	path := "account/getorderhistory"
	if market != "all" {
		path += "?market=" + market
	}
	data, err = b.client.do("GET", 1, path, "", true)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	err = json.Unmarshal(resp.Result, &fills)
	return
}

func (b *Bittrex1) GetOrder(uuid string) (order OrderEx, err error) {
	var (
		data []byte
		resp Response
	)
	const (
		path = "account/getorder"
	)
	data, err = b.client.do("GET", 1, fmt.Sprintf("%s?uuid=%s", path, uuid), "", true)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	if err = handleErr(path, resp); err != nil {
		return
	}
	err = json.Unmarshal(resp.Result, &order)
	return
}

// Bittrex2 represent a v2 Bittrex client
type Bittrex2 struct {
	client *client
}

func (b *Bittrex2) TradeSell(marketName string, orderType string, quantity float64, rate float64, timeInEffect string, conditionType string, target float64) (*Order2, error) {
	var err error

	now := time.Now().UnixNano()

	path := "key/market/TradeSell"

	var params = map[string]string{
		"marketName":    marketName,
		"orderType":     orderType,
		"quantity":      strconv.FormatFloat(quantity, 'f', -1, 64),
		"rate":          strconv.FormatFloat(rate, 'f', -1, 64),
		"timeInEffect":  timeInEffect,
		"conditionType": conditionType,
		"target":        strconv.FormatFloat(target, 'f', -1, 64),
		"_":             strconv.FormatInt(now, 10),
	}

	var qry = false
	for key, value := range params {
		if !qry {
			path = path + "?"
			qry = true
		} else {
			path = path + "&"
		}
		path = path + key + "=" + value
	}

	var data []byte
	if data, err = b.client.do("GET", 2, path, "", true); err != nil {
		return nil, err
	}

	var resp Response
	if err = json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	if err = handleErr(path, resp); err != nil {
		return nil, err
	}

	var order Order2
	if err = json.Unmarshal(resp.Result, &order); err != nil {
		return nil, err
	}

	return &order, nil
}

// New3 returns a v3 Bittrex struct
func New3(apiKey, apiSecret string, appId string) *Bittrex3 {
	client := NewClient(apiKey, apiSecret)
	return &Bittrex3{client, appId}
}

// Bittrex3 represent a v3 Bittrex client
type Bittrex3 struct {
	client *client
	app_id string
}

func (b *Bittrex3) GetMarkets() (markets []Market3, err error) {
	var data []byte
	if data, err = b.client.do3("GET", "markets", nil, b.app_id, false); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &markets); err != nil {
		return nil, err
	}
	return markets, err
}

func (b *Bittrex3) GetTicker(market string) (*Ticker3, error) {
	var (
		err  error
		data []byte
	)
	if data, err = b.client.do3("GET", fmt.Sprintf("markets/%s/ticker", market), nil, b.app_id, false); err != nil {
		return nil, err
	}
	var out Ticker3
	if err = json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *Bittrex3) GetMarketSummary(market string) (*MarketSummary3, error) {
	var (
		err  error
		data []byte
	)
	if data, err = b.client.do3("GET", fmt.Sprintf("markets/%s/summary", market), nil, b.app_id, false); err != nil {
		return nil, err
	}
	var out MarketSummary3
	if err = json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *Bittrex3) GetOrderBook(market string, depth int) (*OrderBook3, error) {
	if depth > 500 {
		depth = 500
	} else if depth < 1 {
		depth = 1
	}
	var (
		err  error
		data []byte
	)
	if data, err = b.client.do3("GET", fmt.Sprintf("markets/%s/orderbook?depth=%d", market, depth), nil, b.app_id, false); err != nil {
		return nil, err
	}
	var out OrderBook3
	if err = json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *Bittrex3) CreateOrder(
	marketSymbol string,
	direction string,
	orderType string,
	quantity float64,
	limit float64,
	timeInForce string,
) (*Order3, error) {
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
	if data, err = b.client.do3("POST", "orders", payload, b.app_id, true); err != nil {
		return nil, err
	}

	var result Order3
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (b *Bittrex3) CancelOrder(orderId string) (err error) {
	_, err = b.client.do3("DELETE", fmt.Sprintf("orders/%s", orderId), nil, b.app_id, true)
	return err
}

func (b *Bittrex3) GetOpenOrders(market string) (orders Orders3, err error) {
	path := func() string {
		result := "orders/open"
		if market != "" && market != "all" {
			result += "?marketSymbol=" + market

		}
		return result
	}
	var data []byte
	if data, err = b.client.do3("GET", path(), nil, b.app_id, true); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func (b *Bittrex3) GetOrder(orderId string) (*Order3, error) {
	var (
		err  error
		data []byte
	)
	if data, err = b.client.do3("GET", fmt.Sprintf("orders/%s", orderId), nil, b.app_id, true); err != nil {
		return nil, err
	}
	var order Order3
	if err = json.Unmarshal(data, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (b *Bittrex3) GetOrderHistory(market string) (Orders3, error) {
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

	get := func(nextPageToken string) (Orders3, error) {
		var (
			err    error
			data   []byte
			result Orders3
		)
		if data, err = b.client.do3("GET", path(nextPageToken), nil, b.app_id, true); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	var (
		err    error
		page   Orders3
		result Orders3
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
