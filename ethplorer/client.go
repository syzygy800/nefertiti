package ethplorer

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	FREE_KEY = "freekey"
	API_BASE = "http://api.ethplorer.io"
)

var (
	lastRequest   time.Time
	BeforeRequest func(apiKey, path string) error = nil
	AfterRequest  func()                          = nil
)

func init() {
	BeforeRequest = func(apiKey, path string) error {
		elapsed := time.Since(lastRequest)
		if elapsed.Seconds() < (float64(1) / RequestsPerSecond(apiKey)) {
			time.Sleep(time.Duration((float64(time.Second) / RequestsPerSecond(apiKey))) - elapsed)
		}
		return nil
	}
	AfterRequest = func() {
		lastRequest = time.Now()
	}
}

func RequestsPerSecond(apiKey string) float64 {
	if apiKey != FREE_KEY {
		return 10
	} else {
		return 0.5
	}
}

type Client struct {
	Key string
}

func New(apiKey string) *Client {
	return &Client{
		Key: apiKey,
	}
}

func (client *Client) get(url string) ([]byte, error) {
	var err error
	var resp *http.Response
	if resp, err = http.Get(url); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data []byte
	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		return nil, err
	}
	return data, nil
}

func (client *Client) query(path string, params url.Values) ([]byte, error) {
	var err error
	var body []byte

	// parse the ethplorer address
	var endpoint *url.URL
	if endpoint, err = url.Parse(API_BASE); err != nil {
		return nil, err
	}

	// set the endpoint for this request
	endpoint.Path += path
	params.Add("apiKey", client.Key)
	endpoint.RawQuery = params.Encode()

	// satisfy the rate limits
	if err = BeforeRequest(client.Key, path); err != nil {
		return nil, err
	}
	defer func() {
		AfterRequest()
	}()

	if body, err = client.get(endpoint.String()); err != nil {
		return nil, err
	}

	type Error struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	var resp Error
	if json.Unmarshal(body, &resp) == nil {
		if resp.Error.Message != "" {
			return nil, errors.New(resp.Error.Message)
		}
	}

	return body, nil
}

func (client *Client) GetTop(sort Criteria) ([]Top, error) {
	var err error

	params := url.Values{}
	params.Add("criteria", sort.String())

	var body []byte
	if body, err = client.query("/getTop", params); err != nil {
		return nil, err
	}

	type Output struct {
		Tokens []Top `json:"tokens"`
		Totals struct {
			Tokens          int64   `json:"tokens"`
			TokensWithPrice int64   `json:"tokensWithPrice"`
			Cap             float64 `json:"cap"`
			CapPrevious     float64 `json:"capPrevious"`
			Volume24h       float64 `json:"volume24h"`
			VolumePrevious  float64 `json:"volumePrevious"`
		} `json:"totals"`
	}

	var out Output
	if err = json.Unmarshal(body, &out); err != nil {
		return nil, errors.New(err.Error() + ": " + string(body))
	}

	return out.Tokens, nil
}
