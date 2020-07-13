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

type client struct {
	apiKey    string
	apiSecret string
}

// NewClient return a new Bittrex HTTP client
func NewClient(apiKey, apiSecret string) (c *client) {
	return &client{apiKey, apiSecret}
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
		resp, err := http.DefaultClient.Do(req)
		done <- result{resp, err}
	}()
	// Wait for the read or the timeout
	select {
	case r := <-done:
		return r.resp, r.err
	case <-timer.C:
		return nil, errors.New("timeout on reading data from Bittrex API")
	}
}

// do prepare and process HTTP request to Bittrex API
func (c *client) do(method string, version int, resource string, payload string, authNeeded bool) (out []byte, err error) {
	if err = BeforeRequest(resource); err != nil {
		return nil, err
	}
	defer func() {
		AfterRequest()
	}()

	var rawurl string
	if strings.HasPrefix(resource, "http") {
		rawurl = resource
	} else {
		rawurl = fmt.Sprintf("%s/%s/%s", fmtApiBase(version), fmtApiVersion(version), resource)
	}

	var req *http.Request
	if req, err = http.NewRequest(method, rawurl, strings.NewReader(payload)); err != nil {
		return
	}
	if method == "POST" || method == "PUT" {
		req.Header.Add("Content-Type", "application/json;charset=utf-8")
	}
	req.Header.Add("Accept", "application/json")

	// Auth
	if authNeeded {
		if len(c.apiKey) == 0 || len(c.apiSecret) == 0 {
			err = errors.New("You need to set API Key and API Secret to call this method")
			return
		}
		nonce := time.Now().UnixNano()
		query := req.URL.Query()
		query.Set("apikey", c.apiKey)
		query.Set("nonce", fmt.Sprintf("%d", nonce))
		req.URL.RawQuery = query.Encode()
		mac := hmac.New(sha512.New, []byte(c.apiSecret))
		_, err = mac.Write([]byte(req.URL.String()))
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Add("apisign", sig)
	}

	timer := time.NewTimer(DEFAULT_HTTP_CLIENT_TIMEOUT * time.Second)

	var resp *http.Response
	if resp, err = c.doTimeoutRequest(timer, req); err != nil {
		return
	}
	defer resp.Body.Close()

	if out, err = ioutil.ReadAll(resp.Body); err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = errors.New(resp.Status)
		if resp.StatusCode == http.StatusTooManyRequests {
			HandleRateLimitErr(resource)
		}
	}

	return
}

func (client *client) do3(method string, path string, payload []byte, app_id string, auth bool) (out []byte, err error) {
	if err = BeforeRequest(path); err != nil {
		return nil, err
	}
	defer func() {
		AfterRequest()
	}()

	var url string
	if strings.HasPrefix(path, "http") {
		url = path
	} else {
		url = fmt.Sprintf("%s/%s/%s", fmtApiBase(3), fmtApiVersion(3), path)
	}

	var req *http.Request
	if req, err = http.NewRequest(method, url, bytes.NewReader(payload)); err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json")

	if app_id != "" {
		req.Header.Add("Application-Id", app_id)
	}

	if auth {
		if len(client.apiKey) == 0 || len(client.apiSecret) == 0 {
			err = errors.New("You need to set API Key and API Secret to call this method")
			return
		}

		// Unix timestamp in millisecond format
		nonce := strconv.FormatInt((time.Now().UnixNano() / int64(time.Millisecond/time.Nanosecond)), 10)

		req.Header.Add("Api-Key", client.apiKey)
		req.Header.Add("Api-Timestamp", nonce)

		hash := sha512.New()
		if _, err = hash.Write([]byte(payload)); err != nil {
			return
		}
		content := hex.EncodeToString(hash.Sum(nil))
		req.Header.Add("Api-Content-Hash", content)

		mac := hmac.New(sha512.New, []byte(client.apiSecret))
		if _, err = mac.Write([]byte(nonce + url + method + content)); err != nil {
			return
		}
		req.Header.Add("Api-Signature", hex.EncodeToString(mac.Sum(nil)))
	}

	timer := time.NewTimer(DEFAULT_HTTP_CLIENT_TIMEOUT * time.Second)

	var resp *http.Response
	if resp, err = client.doTimeoutRequest(timer, req); err != nil {
		return
	}
	defer resp.Body.Close()

	if out, err = ioutil.ReadAll(resp.Body); err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		pair := make(map[string]string)
		json.Unmarshal(out, &pair)
		if msg, ok := pair["code"]; ok {
			return nil, errors.New(msg)
		}
		err = errors.New(resp.Status)
	}

	return
}
