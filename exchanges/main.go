package exchanges

import (
	"strconv"
	"strings"

	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/passphrase"
	"github.com/go-errors/errors"
)

type Exchanges []model.Exchange

func (exchanges *Exchanges) FindByName(name string) model.Exchange {
	for _, exchange := range *exchanges {
		if exchange.GetInfo().Equals(name) {
			return exchange
		}
	}
	return nil
}

func New() *Exchanges {
	var out Exchanges
	out = append(out, NewGdax())
	out = append(out, NewBittrex())
	out = append(out, NewBitstamp())
	out = append(out, NewCexIo())
	out = append(out, NewBinance())
	out = append(out, NewBinanceUS())
	out = append(out, NewHitBTC())
	out = append(out, NewKucoin())
	out = append(out, NewCryptoDotCom())
	return &out
}

func getPrecFromStr(value string, def int) int {
	i := strings.Index(value, ".")
	if i > -1 {
		n := i + 1
		for n < len(value) {
			if string(value[n]) != "0" {
				return n - i
			}
			n++
		}
		return 0
	}
	i, err := strconv.Atoi(value)
	if err == nil && i == 1 {
		return 0
	} else {
		return def
	}
}

func promptForApiKeys(exchange string) (apiKey, apiSecret string, err error) {
	apiKey = flag.Get("api-key").String()
	if apiKey == "" {
		if flag.Listen() {
			return "", "", errors.New("missing argument: api-key")
		}
		var data []byte
		if data, err = passphrase.Read(exchange + " API key"); err != nil {
			return "", "", errors.Wrap(err, 1)
		}
		apiKey = string(data)
	}

	apiSecret = flag.Get("api-secret").String()
	if apiSecret == "" {
		if flag.Listen() {
			return "", "", errors.New("missing argument: api-secret")
		}
		var data []byte
		if data, err = passphrase.Read(exchange + " API secret"); err != nil {
			return "", "", errors.Wrap(err, 1)
		}
		apiSecret = string(data)
	}

	return apiKey, apiSecret, nil
}

func promptForApiKeysEx(exchange string) (apiKey, apiSecret, apiPassphrase string, err error) {
	apiKey, apiSecret, err = promptForApiKeys(exchange)

	if err != nil {
		return apiKey, apiSecret, "", err
	}

	apiPassphrase = flag.Get("api-passphrase").String()
	if apiPassphrase == "" {
		if flag.Listen() {
			return "", "", "", errors.New("missing argument: api-passphrase")
		}
		var data []byte
		if data, err = passphrase.Read(exchange + " API passphrase"); err != nil {
			return "", "", "", errors.Wrap(err, 1)
		}
		apiPassphrase = string(data)
	}

	return apiKey, apiSecret, apiPassphrase, nil
}
