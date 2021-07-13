package cexio

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	Endpoint          = "https://cex.io/api/"
	RequestsPerSecond = 1 // limited to 600 requests per 10 minutes
)

var (
	lastRequest   time.Time
	BeforeRequest func(path string) error = nil
	AfterRequest  func()                  = nil
)

func init() {
	BeforeRequest = func(path string) error {
		elapsed := time.Since(lastRequest)
		if elapsed.Seconds() < (float64(1) / float64(RequestsPerSecond)) {
			time.Sleep((time.Second / RequestsPerSecond) - elapsed)
		}
		return nil
	}
	AfterRequest = func() {
		lastRequest = time.Now()
	}
}

type Client struct {
	URL        string
	Key        string
	Secret     string
	UserName   string
	httpClient *http.Client
}

func New(apiKey, apiSecret, userName string) *Client {
	return &Client{
		URL:      Endpoint,
		Key:      apiKey,
		Secret:   apiSecret,
		UserName: userName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (client *Client) hmac256(message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}

func (client *Client) signature() (string, string) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	message := nonce + client.UserName + client.Key
	signature := client.hmac256(message, client.Secret)
	return signature, nonce
}

func (client *Client) get(url string) ([]byte, error) {
	var err error
	var resp *http.Response
	if resp, err = client.httpClient.Get(url); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data []byte
	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		return nil, err
	}
	return data, nil
}

func (client *Client) post(url string, v url.Values) ([]byte, error) {
	var err error
	var resp *http.Response
	if resp, err = client.httpClient.PostForm(url, v); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data []byte
	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		return nil, err
	}
	return data, nil
}

func (client *Client) query(path string, params map[string]string, private bool) ([]byte, error) {
	var err error
	var body []byte

	// parse the cex.io URL
	var endpoint *url.URL
	if endpoint, err = url.Parse(client.URL); err != nil {
		return nil, err
	}

	// set the endpoint for this request
	endpoint.Path += path

	// satisfy the CEX.IO rate limiter
	if err = BeforeRequest(path); err != nil {
		return nil, err
	}
	defer func() {
		AfterRequest()
	}()

	if !private {
		if body, err = client.get(endpoint.String()); err != nil {
			return nil, err
		}
	} else {
		v := url.Values{}
		// add required key, signature & nonce to values
		signature, nonce := client.signature()
		v.Set("key", client.Key)
		v.Set("signature", signature)
		v.Set("nonce", nonce)
		// add custom params to values
		for key, value := range params {
			v.Add(key, value)
		}
		v.Encode()
		if body, err = client.post(endpoint.String(), v); err != nil {
			return nil, err
		}
	}

	// is this an error?
	resp := make(map[string]string)
	json.Unmarshal(body, &resp)
	if msg, ok := resp["error"]; ok {
		return nil, errors.New(msg)
	}

	return body, nil
}

func (client *Client) CurrencyLimits() ([]Pair, error) {
	var err error

	var body []byte
	if body, err = client.query("currency_limits", nil, false); err != nil {
		return nil, err
	}

	type Output struct {
		OK   string `json:"ok"`
		Data struct {
			Pairs []Pair `json:"pairs"`
		} `json:"data"`
	}

	var output Output
	if err = json.Unmarshal(body, &output); err != nil {
		return nil, errors.New(err.Error() + ": " + string(body))
	}

	return output.Data.Pairs, nil
}

func (client *Client) OrderBook(symbol1, symbol2 string) (*OrderBook, error) {
	var err error

	var body []byte
	if body, err = client.query(fmt.Sprintf("order_book/%s/%s", symbol1, symbol2), nil, false); err != nil {
		return nil, err
	}

	var output OrderBook
	if err = json.Unmarshal(body, &output); err != nil {
		return nil, errors.New(err.Error() + ": " + string(body))
	}

	return &output, nil
}

func (client *Client) Ticker(symbol1, symbol2 string) (*Ticker, error) {
	var err error

	var body []byte
	if body, err = client.query(fmt.Sprintf("ticker/%s/%s", symbol1, symbol2), nil, false); err != nil {
		return nil, err
	}

	var output Ticker
	if err = json.Unmarshal(body, &output); err != nil {
		return nil, errors.New(err.Error() + ": " + string(body))
	}

	return &output, nil
}

func (client *Client) PlaceOrder(symbol1, symbol2 string, side Side, amount, price float64) (*Order, error) {
	var err error

	var params = map[string]string{
		"type":   SideString[side],
		"amount": strconv.FormatFloat(amount, 'f', -1, 64),
		"price":  strconv.FormatFloat(price, 'f', -1, 64),
	}

	var body []byte
	if body, err = client.query(fmt.Sprintf("place_order/%s/%s", symbol1, symbol2), params, true); err != nil {
		return nil, err
	}

	var output Order
	if err = json.Unmarshal(body, &output); err != nil {
		return nil, errors.New(err.Error() + ": " + string(body))
	}

	return &output, nil
}

func (client *Client) PlaceMarketOrder(symbol1, symbol2 string, side Side, amount float64) (*Order, error) {
	var err error

	var params = map[string]string{
		"type":       SideString[side],
		"amount":     strconv.FormatFloat(amount, 'f', -1, 64),
		"order_type": "market",
	}

	var body []byte
	if body, err = client.query(fmt.Sprintf("place_order/%s/%s", symbol1, symbol2), params, true); err != nil {
		return nil, err
	}

	var output Order
	if err = json.Unmarshal(body, &output); err != nil {
		return nil, errors.New(err.Error() + ": " + string(body))
	}

	return &output, nil
}

func (client *Client) OpenOrdersAll() ([]Order, error) {
	var err error

	var body []byte
	if body, err = client.query("open_orders/", nil, true); err != nil {
		return nil, err
	}

	var output []Order
	if err = json.Unmarshal(body, &output); err != nil {
		return nil, errors.New(err.Error() + ": " + string(body))
	}

	return output, nil
}

func (client *Client) OpenOrders(symbol1, symbol2 string) ([]Order, error) {
	var err error

	var body []byte
	if body, err = client.query(fmt.Sprintf("open_orders/%s/%s", symbol1, symbol2), nil, true); err != nil {
		return nil, err
	}

	var output []Order
	if err = json.Unmarshal(body, &output); err != nil {
		return nil, errors.New(err.Error() + ": " + string(body))
	}

	return output, nil
}

func (client *Client) ArchivedOrders(symbol1, symbol2 string) ([]Order, error) {
	var err error

	now := time.Now()

	var params = map[string]string{
		"dateFrom": strconv.FormatInt(now.AddDate(0, -12, 0).Unix(), 10), // from one year ago
		"dateTo":   strconv.FormatInt(now.Unix(), 10),                    // end date: today
		"status":   "d",                                                  // done (fully executed)
	}

	var body []byte
	if body, err = client.query(fmt.Sprintf("archived_orders/%s/%s", symbol1, symbol2), params, true); err != nil {
		return nil, err
	}

	var output []Order
	if err = json.Unmarshal(body, &output); err != nil {
		return nil, errors.New(err.Error() + ": " + string(body))
	}

	return output, nil
}

func (client *Client) ArchivedOrdersAll() ([]Order, error) {
	var (
		err    error
		pairs  []Pair
		result []Order
	)
	if pairs, err = client.CurrencyLimits(); err != nil {
		return nil, err
	}
	for _, pair := range pairs {
		var orders []Order
		if orders, err = client.ArchivedOrders(pair.Symbol1, pair.Symbol2); err != nil {
			return nil, err
		}
		for _, order := range orders {
			result = append(result, order)
		}
	}
	return result, nil
}

func (client *Client) CancelOrder(id string) error {
	var err error

	var params = map[string]string{
		"id": id,
	}

	if _, err = client.query("cancel_order/", params, true); err != nil {
		return err
	}

	return nil
}
