package woo

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	lastRequest   time.Time
	BeforeRequest func(method, path string, rps float64) error = nil
	AfterRequest  func()                                       = nil
)

func init() {
	BeforeRequest = func(method, path string, rps float64) error {
		elapsed := time.Since(lastRequest)
		if elapsed.Seconds() < (float64(1) / rps) {
			time.Sleep(time.Duration((float64(time.Second) / rps) - float64(elapsed)))
		}
		return nil
	}
	AfterRequest = func() {
		lastRequest = time.Now()
	}
}

type Client struct {
	URL        string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

func New(URL, apiKey, apiSecret string) *Client {
	return &Client{
		URL,
		apiKey,
		apiSecret,
		&http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (client *Client) do(req *http.Request) ([]byte, error) {
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		if err, msg := IsError(body); err {
			return body, errors.New(msg)
		}
		return body, errors.New(resp.Status)
	}

	return body, nil
}

func (client *Client) get(path string, query url.Values, auth bool, rps float64) ([]byte, error) {
	// respect the rate limit
	err := BeforeRequest(http.MethodGet, path, rps)
	if err != nil {
		return nil, err
	}
	defer func() {
		AfterRequest()
	}()

	// set the endpoint for this request
	endpoint, err := url.Parse(client.URL)
	if err != nil {
		return nil, err
	}
	endpoint.Path += path
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}

	// create the request
	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	// add autentication headers
	if auth {
		timestamp := time.Now().UnixNano() / 1000000
		req.Header.Add("x-api-key", client.apiKey)
		req.Header.Add("x-api-signature", signature(client.apiSecret, query, timestamp))
		req.Header.Add("x-api-timestamp", strconv.FormatInt(timestamp, 10))
	}

	// do the request
	return client.do(req)
}

func (client *Client) call(method, path string, params url.Values, rps float64) ([]byte, error) {
	// respect the rate limit
	err := BeforeRequest(method, path, rps)
	if err != nil {
		return nil, err
	}
	defer func() {
		AfterRequest()
	}()

	// set the endpoint for this request
	endpoint, err := url.Parse(client.URL)
	if err != nil {
		return nil, err
	}
	endpoint.Path += path

	// create the request
	payload := func() string {
		if params != nil {
			return params.Encode()
		}
		return ""
	}()
	req, err := http.NewRequest(method, endpoint.String(), func() io.Reader {
		if payload != "" {
			return strings.NewReader(payload)
		}
		return nil
	}())
	if err != nil {
		return nil, err
	}

	// add autentication headers
	timestamp := time.Now().UnixNano() / 1000000
	req.Header.Add("x-api-key", client.apiKey)
	req.Header.Add("x-api-signature", signature(client.apiSecret, params, timestamp))
	req.Header.Add("x-api-timestamp", strconv.FormatInt(timestamp, 10))
	if payload != "" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	// do the request
	return client.do(req)
}
