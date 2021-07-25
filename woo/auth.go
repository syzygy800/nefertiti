package woo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
)

// sign a request, returns x-api-signature
func signature(apiSecret string, params url.Values, timestamp int64) string {
	// sort the params alphabetically and format them as a query string
	normalized := func() string {
		if params != nil {
			return params.Encode()
		}
		return ""
	}()
	// concat timestamp with above result, using | as seperator
	normalized = normalized + "|" + strconv.FormatInt(timestamp, 10)
	// hash with HMAC SHA256 algorithm
	hash := hmac.New(sha256.New, []byte(apiSecret))
	hash.Write([]byte(normalized))
	return hex.EncodeToString(hash.Sum(nil))
}
