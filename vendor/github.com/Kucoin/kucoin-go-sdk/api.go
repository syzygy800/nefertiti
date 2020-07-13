/*
Package kucoin provides two kinds of APIs: `RESTful API` and `WebSocket feed`.
The official document: https://docs.kucoin.com
*/
package kucoin

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const (
	REQUESTS_PER_SECOND = 30 // 1800 reqs per minute
)

var (
	lastRequest   time.Time
	BeforeRequest func(client *ApiService, request *Request) error = nil
	AfterRequest  func()                                           = nil
)

func RequestsPerSecond(request *Request) float64 {
	if request.Path == "/api/v1/fills" { // list fills
		return 10 // 100 reqs per 10 seconds
	}
	if request.Path == "/api/v1/limit/fills" { // recent fills
		return 10 // 100 reqs per 10 seconds
	}
	if request.Path == "/api/v1/orders" { // list orders
		return 10 // 100 reqs per 10 seconds
	}
	if request.Path == "/api/v1/limit/orders" { // recent orders
		return 10 // 100 reqs per 10 seconds
	}
	return REQUESTS_PER_SECOND
}

func init() {
	BeforeRequest = func(client *ApiService, request *Request) error {
		elapsed := time.Since(lastRequest)
		rps := RequestsPerSecond(request)
		if elapsed.Seconds() < (float64(1) / rps) {
			time.Sleep(time.Duration((float64(time.Second) / rps)) - elapsed)
		}
		return nil
	}
	AfterRequest = func() {
		lastRequest = time.Now()
	}
}

// An ApiService provides a HTTP client and a signer to make a HTTP request with the signature to KuCoin API.
type ApiService struct {
	apiBaseURI       string
	apiKey           string
	apiSecret        string
	apiPassphrase    string
	apiSkipVerifyTls bool
	requester        Requester
	signer           Signer
}

// ProductionApiBaseURI is api base uri for production.
const ProductionApiBaseURI = "https://api.kucoin.com"

// An ApiServiceOption is a option parameter to create the instance of ApiService.
type ApiServiceOption func(service *ApiService)

// ApiBaseURIOption creates a instance of ApiServiceOption about apiBaseURI.
func ApiBaseURIOption(uri string) ApiServiceOption {
	return func(service *ApiService) {
		service.apiBaseURI = uri
	}
}

// ApiKeyOption creates a instance of ApiServiceOption about apiKey.
func ApiKeyOption(key string) ApiServiceOption {
	return func(service *ApiService) {
		service.apiKey = key
	}
}

// ApiSecretOption creates a instance of ApiServiceOption about apiSecret.
func ApiSecretOption(secret string) ApiServiceOption {
	return func(service *ApiService) {
		service.apiSecret = secret
	}
}

// ApiPassPhraseOption creates a instance of ApiServiceOption about apiPassPhrase.
func ApiPassPhraseOption(passPhrase string) ApiServiceOption {
	return func(service *ApiService) {
		service.apiPassphrase = passPhrase
	}
}

// ApiSkipVerifyTlsOption creates a instance of ApiServiceOption about apiSkipVerifyTls.
func ApiSkipVerifyTlsOption(skipVerifyTls bool) ApiServiceOption {
	return func(service *ApiService) {
		service.apiSkipVerifyTls = skipVerifyTls
	}
}

// NewApiService creates a instance of ApiService by passing ApiServiceOptions, then you can call methods.
func NewApiService(opts ...ApiServiceOption) *ApiService {
	as := &ApiService{
		requester: &BasicRequester{},
	}
	for _, opt := range opts {
		opt(as)
	}
	if as.apiBaseURI == "" {
		as.apiBaseURI = ProductionApiBaseURI
	}
	if as.apiKey != "" {
		as.signer = NewKcSigner(as.apiKey, as.apiSecret, as.apiPassphrase)
	}
	return as
}

// NewApiServiceFromEnv creates a instance of ApiService by environmental variables such as `API_BASE_URI` `API_KEY` `API_SECRET` `API_PASSPHRASE`, then you can call the methods of ApiService.
func NewApiServiceFromEnv() *ApiService {
	return NewApiService(
		ApiBaseURIOption(os.Getenv("API_BASE_URI")),
		ApiKeyOption(os.Getenv("API_KEY")),
		ApiSecretOption(os.Getenv("API_SECRET")),
		ApiPassPhraseOption(os.Getenv("API_PASSPHRASE")),
		ApiSkipVerifyTlsOption(os.Getenv("API_SKIP_VERIFY_TLS") == "1"),
	)
}

// Call calls the API by passing *Request and returns *ApiResponse.
func (as *ApiService) Call(request *Request) (*ApiResponse, error) {
	// --- BEGIN --- svanas 2019-02-13 --- satisfy the rate limiter -------
	if err := BeforeRequest(as, request); err != nil {
		return nil, err
	}
	defer func() {
		AfterRequest()
		// ---- END ---- svanas 2019-02-13 --- satisfy the rate limiter ---
		if err := recover(); err != nil {
			log.Println("[[Recovery] panic recovered:", err)
		}
	}()

	request.BaseURI = as.apiBaseURI
	request.SkipVerifyTls = as.apiSkipVerifyTls
	request.Header.Set("Content-Type", "application/json")
	if as.signer != nil {
		var b bytes.Buffer
		b.WriteString(request.Method)
		b.WriteString(request.RequestURI())
		b.Write(request.Body)
		h := as.signer.(*KcSigner).Headers(b.String())
		for k, v := range h {
			request.Header.Set(k, v)
		}
	}

	var (
		err error
		rsp *Response
	)
	// --- BEGIN --- svanas 2020-02-18 --- no such host? wait a few secs, then try again. for a max of 10 attempts ----
	attempts := 0
	for {
		rsp, err = as.requester.Request(request, request.Timeout)
		if err == nil {
			break
		} else {
			if strings.Contains(err.Error(), "no such host") {
				attempts++
				if attempts >= 10 {
					return nil, err
				} else {
					time.Sleep(6 * time.Second)
				}
			} else {
				return nil, err
			}
		}
	}
	// ---- END ---- svanas 2020-02-18 --------------------------------------------------------------------------------

	ar := &ApiResponse{response: rsp}
	if err := rsp.ReadJsonBody(ar); err != nil {
		rb, _ := rsp.ReadBody()
		m := fmt.Sprintf("[Parse]Failure: parse JSON body failed because %s, %s %s with body=%s, respond code=%d body=%s",
			err.Error(),
			rsp.request.Method,
			rsp.request.RequestURI(),
			string(rsp.request.Body),
			rsp.StatusCode,
			string(rb),
		)
		return ar, errors.New(m)
	}
	return ar, nil
}
