// Package Bittrex is an implementation of the Bittrex API in Golang.
package bittrex

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	TIME_FORMAT = time.RFC3339
	API_BASE    = "https://api.bittrex.com"
	API_VERSION = "v3"
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

func getRequestsPerSecond(path string) (float64, bool) { // -> (rps, cooldown)
	if cooldown {
		cooldown = false
		return RequestsPerSecond(INTENSITY_SUPER), true
	}
	for i := range path {
		if strings.Contains("?", string(path[i])) {
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
		rps, cooled := getRequestsPerSecond(path)
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
			if strings.Contains("?", string(path[idx])) {
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

type Client struct {
	apiKey     string
	apiSecret  string
	appId      string
	httpClient *http.Client
}

func New(apiKey, apiSecret string, appId string) *Client {
	return &Client{
		apiKey,
		apiSecret,
		appId,
		&http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (client *Client) do(method string, path string, payload []byte, auth bool) ([]byte, error) {
	var (
		code int
		out  []byte
		err  error
	)
	for {
		code, out, err = client._do(method, path, payload, auth)
		if code != http.StatusTooManyRequests {
			break
		}
	}
	return out, err
}

func (client *Client) _do(method string, path string, payload []byte, auth bool) (int, []byte, error) {
	var (
		err    error
		out    []byte
		cooled bool = false
	)

	if cooled, err = BeforeRequest(path); err != nil {
		return 0, nil, err
	}
	defer func() {
		AfterRequest()
	}()

	var url string
	if strings.HasPrefix(path, "http") {
		url = path
	} else {
		url = fmt.Sprintf("%s/%s/%s", API_BASE, API_VERSION, path)
	}

	var req *http.Request
	if req, err = http.NewRequest(method, url, bytes.NewReader(payload)); err != nil {
		return 0, nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	if client.appId != "" {
		req.Header.Add("Application-Id", client.appId)
	}

	if auth {
		if len(client.apiKey) == 0 || len(client.apiSecret) == 0 {
			err = errors.New("you need to set API key and API secret to call this method")
			return 0, nil, err
		}

		// Unix timestamp in millisecond format
		nonce := strconv.FormatInt((time.Now().UnixNano() / int64(time.Millisecond/time.Nanosecond)), 10)

		req.Header.Add("Api-Key", client.apiKey)
		req.Header.Add("Api-Timestamp", nonce)

		hash := sha512.New()
		if _, err = hash.Write([]byte(payload)); err != nil {
			return 0, nil, err
		}
		content := hex.EncodeToString(hash.Sum(nil))
		req.Header.Add("Api-Content-Hash", content)

		mac := hmac.New(sha512.New, []byte(client.apiSecret))
		if _, err = mac.Write([]byte(nonce + url + method + content)); err != nil {
			return 0, nil, err
		}
		req.Header.Add("Api-Signature", hex.EncodeToString(mac.Sum(nil)))
	}

	var resp *http.Response
	if resp, err = client.httpClient.Do(req); err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if out, err = ioutil.ReadAll(resp.Body); err != nil {
		return resp.StatusCode, nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		pair := make(map[string]string)
		json.Unmarshal(out, &pair)
		if msg, ok := pair["code"]; ok {
			err = errors.New(msg)
		} else {
			err = errors.New(resp.Status)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			if HandleRateLimitErr(path, cooled) != nil {
				return 0, nil, err
			}
		}
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, out, nil
}

func (client *Client) GetMarkets() (markets []Market, err error) {
	var data []byte
	if data, err = client.do("GET", "markets", nil, false); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &markets); err != nil {
		return nil, err
	}
	return markets, err
}

func (client *Client) GetTicker(market string) (*Ticker, error) {
	var (
		err  error
		data []byte
	)
	if data, err = client.do("GET", fmt.Sprintf("markets/%s/ticker", market), nil, false); err != nil {
		return nil, err
	}
	var out Ticker
	if err = json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (client *Client) GetMarketSummary(market string) (*MarketSummary, error) {
	var (
		err  error
		data []byte
	)
	if data, err = client.do("GET", fmt.Sprintf("markets/%s/summary", market), nil, false); err != nil {
		return nil, err
	}
	var out MarketSummary
	if err = json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (client *Client) GetOrderBook(market string, depth int) (*OrderBook, error) {
	if depth > 500 {
		depth = 500
	} else if depth < 1 {
		depth = 1
	}
	var (
		err  error
		data []byte
	)
	if data, err = client.do("GET", fmt.Sprintf("markets/%s/orderbook?depth=%d", market, depth), nil, false); err != nil {
		return nil, err
	}
	var out OrderBook
	if err = json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (client *Client) CreateOrder(
	marketSymbol string,
	direction OrderSide,
	orderType OrderType,
	quantity float64,
	limit float64,
	timeInForce TimeInForce,
) (*Order, error) {
	var err error

	order := &newOrder{
		MarketSymbol: marketSymbol,
		Direction:    direction.String(),
		OrderType:    orderType.String(),
		Quantity:     strconv.FormatFloat(quantity, 'f', -1, 64),
		TimeInForce:  timeInForce.String(),
	}

	if limit > 0 {
		order.Limit = strconv.FormatFloat(limit, 'f', -1, 64)
	}

	var payload []byte
	if payload, err = json.Marshal(order); err != nil {
		return nil, err
	}

	var data []byte
	if data, err = client.do("POST", "orders", payload, true); err != nil {
		return nil, err
	}

	var result Order
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (client *Client) CancelOrder(orderId OrderId) (err error) {
	_, err = client.do("DELETE", fmt.Sprintf("orders/%s", orderId), nil, true)
	return err
}

func (client *Client) GetOpenOrders(market string) (orders Orders, err error) {
	var data []byte
	if data, err = client.do("GET", func() string {
		result := "orders/open"
		if market != "" && market != "all" {
			result += "?marketSymbol=" + market

		}
		return result
	}(), nil, true); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func (client *Client) GetOrder(orderId OrderId) (*Order, error) {
	var (
		err  error
		data []byte
	)
	if data, err = client.do("GET", fmt.Sprintf("orders/%s", orderId), nil, true); err != nil {
		return nil, err
	}
	var order Order
	if err = json.Unmarshal(data, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (client *Client) GetOrderHistory(market string) (Orders, error) {
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
		if data, err = client.do("GET", path(nextPageToken), nil, true); err != nil {
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
		if page, err = get(string(page[len(page)-1].Id)); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (client *Client) CreateConditionalOrder(
	marketSymbol string,
	operand Operand,
	triggerPrice float64,
	orderToCreate *NewOrder,
	orderToCancel OrderId,
) (*ConditionalOrder, error) {
	var err error

	type newConditionalOrder struct {
		MarketSymbol  string                     `json:"marketSymbol"`
		Operand       string                     `json:"operand"` // LTE or GTE
		TriggerPrice  string                     `json:"triggerPrice"`
		OrderToCreate *newOrder                  `json:"orderToCreate"`
		OrderToCancel *newCancelConditionalOrder `json:"orderToCancel"`
	}

	order := &newConditionalOrder{
		MarketSymbol: marketSymbol,
		Operand:      operand.String(),
		TriggerPrice: strconv.FormatFloat(triggerPrice, 'f', -1, 64),
	}

	if orderToCreate != nil {
		helper := orderToCreate.into()
		order.OrderToCreate = &helper
	}

	if orderToCancel != "" {
		order.OrderToCancel = &newCancelConditionalOrder{
			OrderType: "ORDER",
			Id:        orderToCancel,
		}
	}

	var payload []byte
	if payload, err = json.Marshal(order); err != nil {
		return nil, err
	}

	var data []byte
	if data, err = client.do("POST", "conditional-orders", payload, true); err != nil {
		return nil, err
	}

	var result ConditionalOrder
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (client *Client) CancelConditionalOrder(orderId OrderId) (err error) {
	_, err = client.do("DELETE", fmt.Sprintf("conditional-orders/%s", orderId), nil, true)
	return err
}

func (client *Client) GetOpenConditionalOrders(market string) (orders []ConditionalOrder, err error) {
	var data []byte
	if data, err = client.do("GET", func() string {
		result := "conditional-orders/open"
		if market != "" && market != "all" {
			result += "?marketSymbol=" + market
		}
		return result
	}(), nil, true); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}
