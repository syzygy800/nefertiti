package kucoin

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	requestsPerSecond = 30 // 1800 reqs per minute
)

var (
	lastRequest   time.Time
	BeforeRequest func(client *ApiService, request *Request, rps float64) error = nil
	AfterRequest  func()                                                        = nil
)

func init() {
	BeforeRequest = func(client *ApiService, request *Request, rps float64) error {
		elapsed := time.Since(lastRequest)
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
	apiBaseURI    string
	apiKey        string
	apiSecret     string
	apiPassphrase string
	requester     Requester
	signer        Signer
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
		as.signer = NewKcSigner(as.apiKey, as.apiSecret, as.apiPassphrase, 2)
	}
	return as
}

// Call calls the API by passing *Request and returns *ApiResponse.
func (as *ApiService) call(request *Request, rps float64) (*ApiResponse, error) {
	// --- BEGIN --- svanas 2019-02-13 --- satisfy the rate limiter -------
	if err := BeforeRequest(as, request, rps); err != nil {
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
	// --- BEGIN --- svanas 2020-02-18 --- no such host? wait a sec, then try again. for a max of 10 attempts ----
	for i := 0; i < 10; i++ {
		rsp, err = as.requester.Request(request, request.Timeout)
		// --- BEGIN --- svanas 2021-07-31 --- rate limit is exceeded? cool down for 10 seconds ------------------
		if rsp != nil && rsp.StatusCode == http.StatusTooManyRequests {
			time.Sleep(10 * time.Second)
		} else
		// ---- END ---- svanas 2021-07-31 -----------------------------------------------------------------------
		if err == nil {
			break
		} else if strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "network is unreachable") {
			time.Sleep(time.Second)
		} else {
			return nil, err
		}
	}
	// ---- END ---- svanas 2020-02-18 ---------------------------------------------------------------------------
	if err != nil {
		return nil, err
	}

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
