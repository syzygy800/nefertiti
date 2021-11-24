package huobi

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

var (
	lastRequest       time.Time
	RequestsPerSecond float64                         = 10
	BeforeRequest     func(method, path string) error = nil
	AfterRequest      func()                          = nil
)

func init() {
	BeforeRequest = func(method, path string) error {
		elapsed := time.Since(lastRequest)
		if elapsed.Seconds() < (float64(1) / RequestsPerSecond) {
			time.Sleep(time.Duration((float64(time.Second) / RequestsPerSecond) - float64(elapsed)))
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
		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			return nil, errors.New(resp.Status)
		}
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

func (client *Client) get(path string, query url.Values) ([]byte, error) {
	// respect the rate limit
	err := BeforeRequest(http.MethodGet, path)
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

	// do the request
	return client.do(req)
}
