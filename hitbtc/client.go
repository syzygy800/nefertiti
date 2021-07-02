package hitbtc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type client struct {
	apiKey      string
	apiSecret   string
	httpClient  *http.Client
	httpTimeout time.Duration
}

const (
	httpClientTimeout = 30
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

// NewClient return a new HitBtc HTTP client
func NewClient(apiKey, apiSecret string) (c *client) {
	return &client{apiKey, apiSecret, &http.Client{}, httpClientTimeout * time.Second}
}

// NewClientWithCustomHttpConfig returns a new HitBtc HTTP client using the predefined http client
func NewClientWithCustomHttpConfig(apiKey, apiSecret string, httpClient *http.Client) (c *client) {
	timeout := httpClient.Timeout
	if timeout <= 0 {
		timeout = httpClientTimeout * time.Second
	}
	return &client{apiKey, apiSecret, httpClient, timeout}
}

// NewClient returns a new HitBtc HTTP client with custom timeout
func NewClientWithCustomTimeout(apiKey, apiSecret string, timeout time.Duration) (c *client) {
	return &client{apiKey, apiSecret, &http.Client{}, timeout}
}

// doTimeoutRequest do a HTTP request with timeout
func (c *client) doTimeoutRequest(timer *time.Timer, req *http.Request) (*http.Response, error) {
	// Do the request in the background so we can check the timeout
	type result struct {
		resp *http.Response
		err  error
	}
	done := make(chan result, 1)
	go func() {
		resp, err := c.httpClient.Do(req)
		done <- result{resp, err}
	}()
	// Wait for the read or the timeout
	select {
	case r := <-done:
		return r.resp, r.err
	case <-timer.C:
		return nil, errors.New("timeout on reading data from HitBtc API")
	}
}

// do prepare and process HTTP request to HitBtc API
func (c *client) do(method string, resource string, payload map[string]string, authNeeded bool) (response []byte, err error) {
	// the limit is 10 requests per second
	if err = BeforeRequest(method, resource); err != nil {
		return nil, err
	}
	defer func() {
		AfterRequest()
	}()

	connectTimer := time.NewTimer(c.httpTimeout)

	var rawurl string
	if strings.HasPrefix(resource, "http") {
		rawurl = resource
	} else {
		rawurl = fmt.Sprintf("%s/%s", API_BASE, resource)
	}
	var formData string
	if method == "GET" {
		var URL *url.URL
		URL, err = url.Parse(rawurl)
		if err != nil {
			return
		}
		q := URL.Query()
		for key, value := range payload {
			q.Set(key, value)
		}
		formData = q.Encode()
		URL.RawQuery = formData
		rawurl = URL.String()
	} else {
		formValues := url.Values{}
		for key, value := range payload {
			formValues.Set(key, value)
		}
		formData = formValues.Encode()
	}
	req, err := http.NewRequest(method, rawurl, strings.NewReader(formData))
	if err != nil {
		return
	}

	req.Header.Add("Accept", "application/json")
	if method != "GET" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	// Auth
	if authNeeded {
		if len(c.apiKey) == 0 || len(c.apiSecret) == 0 {
			err = errors.New("You need to set API Key and API Secret to call this method")
			return
		}
		req.SetBasicAuth(c.apiKey, c.apiSecret)
	}

	resp, err := c.doTimeoutRequest(connectTimer, req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	response, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 401 {
		//--- BEGIN --- svanas --- 2018-04-04 ---------------------------------
		var body interface{}
		if err = json.Unmarshal(response, &body); err == nil {
			if err = handleErr(body); err != nil {
				return response, err
			}
		}
		//---- END ---- svanas --- 2018-04-04 ---------------------------------
		err = errors.New(resp.Status)
	}
	return response, err
}
