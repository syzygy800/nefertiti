package kucoin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"
)

// KcSigner is the implement of Signer for KuCoin.
type KcSigner struct {
	apiKey           string
	apiSecret        string
	apiPassPhrase    string
	apiPartnerID     string
	apiPartnerSecret string
	apiKeyVersion    int
}

// makes a sha256 signature.
func (ks *KcSigner) sha256(plain, key []byte) []byte {
	hm := hmac.New(sha256.New, key)
	hm.Write(plain)
	return hm.Sum(nil)
}

// make a sha256 signature with `apiKey` `apiSecret` `apiPassPhrase`.
func (ks *KcSigner) sign(plain []byte) []byte {
	s := ks.sha256(plain, []byte(ks.apiSecret))
	return []byte(base64.StdEncoding.EncodeToString(s))
}

func (ks *KcSigner) signPartner(timestamp string) string {
	prehash := timestamp + ks.apiPartnerID + ks.apiKey
	s := ks.sha256([]byte(prehash), []byte(ks.apiPartnerSecret))
	return base64.StdEncoding.EncodeToString(s)
}

// Headers returns a map of signature header.
func (ks *KcSigner) Headers(plain string) map[string]string {
	t := IntToString(time.Now().UnixNano() / 1000000)
	p := []byte(t + plain)
	s := string(ks.sign(p))
	pp := ks.apiPassPhrase
	if ks.apiKeyVersion > 1 {
		pp = string(ks.sign([]byte(ks.apiPassPhrase)))
	}
	return map[string]string{
		"KC-API-KEY":          ks.apiKey,
		"KC-API-PASSPHRASE":   pp,
		"KC-API-TIMESTAMP":    t,
		"KC-API-SIGN":         s,
		"KC-API-KEY-VERSION":  strconv.Itoa(ks.apiKeyVersion),
		"KC-API-PARTNER":      ks.apiPartnerID,
		"KC-API-PARTNER-SIGN": ks.signPartner(t),
	}
}

// NewKcSigner creates a instance of KcSigner.
func NewKcSigner(key, secret, passPhrase, partnerID, partnerSecret string, version int) *KcSigner {
	return &KcSigner{
		apiKey:           key,
		apiSecret:        secret,
		apiPassPhrase:    passPhrase,
		apiPartnerID:     partnerID,
		apiPartnerSecret: partnerSecret,
		apiKeyVersion:    version,
	}
}
