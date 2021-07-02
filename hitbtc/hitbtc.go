// Package HitBTC is an implementation of the HitBTC API in Golang.
package hitbtc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	API_BASE = "https://api.hitbtc.com/api/2" // HitBtc API endpoint
)

// New returns an instantiated HitBTC struct
func New(apiKey, apiSecret string) *HitBtc {
	client := NewClient(apiKey, apiSecret)
	return &HitBtc{client}
}

// NewWithCustomHttpClient returns an instantiated HitBTC struct with custom http client
func NewWithCustomHttpClient(apiKey, apiSecret string, httpClient *http.Client) *HitBtc {
	client := NewClientWithCustomHttpConfig(apiKey, apiSecret, httpClient)
	return &HitBtc{client}
}

// NewWithCustomTimeout returns an instantiated HitBTC struct with custom timeout
func NewWithCustomTimeout(apiKey, apiSecret string, timeout time.Duration) *HitBtc {
	client := NewClientWithCustomTimeout(apiKey, apiSecret, timeout)
	return &HitBtc{client}
}

// handleErr gets JSON response from livecoin API en deal with error
func handleErr(r interface{}) error {
	switch v := r.(type) {
	case map[string]interface{}:
		error := r.(map[string]interface{})["error"]
		if error != nil {
			switch v := error.(type) {
			case map[string]interface{}:
				errorMessage := error.(map[string]interface{})["message"]
				return errors.New(errorMessage.(string))
			default:
				return fmt.Errorf("I don't know about type %T!\n", v)
			}
		}
	case []interface{}:
		return nil
	default:
		return fmt.Errorf("I don't know about type %T!\n", v)
	}

	return nil
}

// HitBtc represent a HitBTC client
type HitBtc struct {
	client *client
}

// GetSymbols is used to get the open and available trading markets at HitBtc along with other meta data.
func (b *HitBtc) GetSymbols() (symbols []Symbol, err error) {
	r, err := b.client.do("GET", "public/symbol", nil, false)
	if err != nil {
		return
	}
	var response interface{}
	if err = json.Unmarshal(r, &response); err != nil {
		return
	}
	if err = handleErr(response); err != nil {
		return
	}
	err = json.Unmarshal(r, &symbols)
	return
}

// GetTicker is used to get the current ticker values for a market.
func (b *HitBtc) GetTicker(market string) (ticker Ticker, err error) {
	r, err := b.client.do("GET", "public/ticker/"+strings.ToUpper(market), nil, false)
	if err != nil {
		return
	}
	var response interface{}
	if err = json.Unmarshal(r, &response); err != nil {
		return
	}
	if err = handleErr(response); err != nil {
		return
	}
	err = json.Unmarshal(r, &ticker)
	return
}

// GetTrades used to retrieve your trade history.
// market string literal for the market (ie. BTC/LTC). If set to "all", will return for all market
func (b *HitBtc) GetTrades(currencyPair string) (trades []Trade, err error) {
	payload := make(map[string]string)
	if currencyPair != "all" {
		payload["symbol"] = currencyPair
	}
	r, err := b.client.do("GET", "history/trades", payload, true)
	if err != nil {
		return
	}
	var response interface{}
	if err = json.Unmarshal(r, &response); err != nil {
		return
	}
	if err = handleErr(response); err != nil {
		return
	}
	err = json.Unmarshal(r, &trades)
	return
}

func (b *HitBtc) CancelOrder(currencyPair string) (orders []Order, err error) {
	payload := make(map[string]string)
	if currencyPair != "all" {
		payload["symbol"] = currencyPair
	}
	r, err := b.client.do("DELETE", "order", payload, true)
	if err != nil {
		return
	}
	var response interface{}
	if err = json.Unmarshal(r, &response); err != nil {
		return
	}
	if err = handleErr(response); err != nil {
		return
	}
	err = json.Unmarshal(r, &orders)
	return
}

func (b *HitBtc) CancelClientOrderId(clientOrderId string) (order Order, err error) {
	r, err := b.client.do("DELETE", "order/"+clientOrderId, nil, true)
	if err != nil {
		return
	}
	var response interface{}
	if err = json.Unmarshal(r, &response); err != nil {
		return
	}
	if err = handleErr(response); err != nil {
		return
	}
	err = json.Unmarshal(r, &order)
	return
}

func (b *HitBtc) GetOrder(orderId string) (orders []Order, err error) {
	payload := make(map[string]string)
	payload["clientOrderId"] = orderId
	r, err := b.client.do("GET", "history/order", payload, true)
	if err != nil {
		return
	}
	var response interface{}
	if err = json.Unmarshal(r, &response); err != nil {
		return
	}
	if err = handleErr(response); err != nil {
		return
	}
	err = json.Unmarshal(r, &orders)
	return
}

func (b *HitBtc) GetOrderHistory() (orders []Order, err error) {
	r, err := b.client.do("GET", "history/order", nil, true)
	if err != nil {
		return
	}
	var response interface{}
	if err = json.Unmarshal(r, &response); err != nil {
		return
	}
	if err = handleErr(response); err != nil {
		return
	}
	err = json.Unmarshal(r, &orders)
	return
}

func (b *HitBtc) GetOpenOrders(currencyPair string) (orders []Order, err error) {
	payload := make(map[string]string)
	if currencyPair != "all" {
		payload["symbol"] = currencyPair
	}
	r, err := b.client.do("GET", "order", payload, true)
	if err != nil {
		return
	}
	var response interface{}
	if err = json.Unmarshal(r, &response); err != nil {
		return
	}
	if err = handleErr(response); err != nil {
		return
	}
	err = json.Unmarshal(r, &orders)
	return
}

func (b *HitBtc) PlaceOrder(
	clientOrderId string,
	symbol string,
	side string,
	orderType string,
	timeInForce string,
	quantity float64,
	price float64,
	stopPrice float64,
) (responseOrder Order, err error) {
	payload := make(map[string]string)

	payload["symbol"] = symbol
	payload["side"] = side
	payload["type"] = orderType
	payload["timeInForce"] = timeInForce
	payload["quantity"] = strconv.FormatFloat(quantity, 'f', -1, 64)
	if price > 0 {
		payload["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	}
	if stopPrice > 0 {
		payload["stopPrice"] = strconv.FormatFloat(stopPrice, 'f', -1, 64)
	}

	var r []byte
	if clientOrderId == "" {
		r, err = b.client.do("POST", "order", payload, true)
	} else {
		r, err = b.client.do("PUT", "order/"+clientOrderId, payload, true)
	}
	if err != nil {
		return
	}
	var response interface{}
	if err = json.Unmarshal(r, &response); err != nil {
		return
	}
	if err = handleErr(response); err != nil {
		return
	}
	err = json.Unmarshal(r, &responseOrder)
	return
}

func (b *HitBtc) GetOrderBook(currencyPair string, limit uint32) (book Book, err error) {
	payload := make(map[string]string)
	if limit != 100 {
		payload["limit"] = strconv.FormatUint(uint64(limit), 10)
	}
	r, err := b.client.do("GET", "public/orderbook/"+currencyPair, payload, false)
	if err != nil {
		return
	}
	var response interface{}
	if err = json.Unmarshal(r, &response); err != nil {
		return
	}
	if err = handleErr(response); err != nil {
		return
	}
	err = json.Unmarshal(r, &book)
	return
}
