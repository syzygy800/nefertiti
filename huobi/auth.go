package huobi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
)

func sign(apiSecret string, method string, host string, path string, params url.Values) string {
	// sort the params alphabetically and format them as a query string
	normalized := func() string {
		if params != nil {
			return params.Encode()
		}
		return ""
	}()
	// concat method and host and path with above result, using line break as seperator
	presigned := method + "\n" + host + "\n" + path + "\n" + normalized
	// hash with HMAC SHA256 algorithm
	hash := hmac.New(sha256.New, []byte(apiSecret))
	hash.Write([]byte(presigned))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}
