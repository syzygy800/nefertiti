package binance

import (
	"context"
	"strconv"
	"strings"

	exchange "github.com/adshao/go-binance/v2"
)

type (
	Prec struct {
		Symbol exchange.Symbol
		Price  int
		Size   int
		Min    float64 // minimum notional value (aka price * quantity) allowed for an order
	}
	Precs []Prec
)

func (self Precs) indexBySymbol(symbol string) int {
	for i, p := range self {
		if p.Symbol.Symbol == symbol {
			return i
		}
	}
	return -1
}

func (self Precs) PrecFromSymbol(symbol string) *Prec {
	i := self.indexBySymbol(symbol)
	if i != -1 {
		return &self[i]
	}
	return nil
}

var cache Precs

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

func getPrecs(client *Client) (Precs, error) {
	var out Precs

	defer AfterRequest()
	BeforeRequest(client, WEIGHT_EXCHANGE_INFO)

	info, err := client.inner.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		return nil, err
	} else {
		for _, symbol := range info.Symbols {
			prec := Prec{
				Symbol: symbol,
			}
			for _, filter := range symbol.Filters {
				if filter["filterType"] == string(exchange.SymbolFilterTypeLotSize) {
					if val, ok := filter["stepSize"]; ok {
						if str, ok := val.(string); ok {
							prec.Size = getPrecFromStr(str, 0)
						}
					}
				}
				if filter["filterType"] == string(exchange.SymbolFilterTypePriceFilter) {
					if val, ok := filter["tickSize"]; ok {
						if str, ok := val.(string); ok {
							prec.Price = getPrecFromStr(str, 8)
						}
					}
				}
				if filter["filterType"] == string(exchange.SymbolFilterTypeMinNotional) {
					if val, ok := filter["minNotional"]; ok {
						if str, ok := val.(string); ok {
							if prec.Min, err = strconv.ParseFloat(str, 64); err != nil {
								return nil, err
							}
						}
					}
				}
			}
			out = append(out, prec)
		}
	}
	return out, nil
}

func GetPrecs(client *Client, cached bool) (Precs, error) {
	if cache == nil || cached == false {
		var err error
		if cache, err = getPrecs(client); err != nil {
			return nil, err
		}
	}
	return cache, nil
}
