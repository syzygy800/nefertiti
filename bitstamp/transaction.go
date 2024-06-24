package bitstamp

import (
	"fmt"
	"math"
	"time"

	"github.com/svanas/nefertiti/any"
	"github.com/svanas/nefertiti/errors"
)

type (
	Transaction map[string]interface{}
)

func (transaction *Transaction) OrderId() string {
	return any.AsString((*transaction)["order_id"])
}

func (transaction *Transaction) Market(client *Client) string {
	cached := true
	for {
		markets, _ := GetMarkets(client, cached)
		for i := range markets {
			price := (*transaction)[fmt.Sprintf("%s_%s", markets[i].Base, markets[i].Quote)]
			if any.AsString(price) != "" {
				return markets[i].Name
			}
		}
		if !cached {
			return ""
		}
		cached = false
	}
}

func (transaction *Transaction) Price(client *Client) float64 {
	cached := true
	for {
		markets, _ := GetMarkets(client, cached)
		for i := range markets {
			price := (*transaction)[fmt.Sprintf("%s_%s", markets[i].Base, markets[i].Quote)]
			if any.AsString(price) != "" {
				return any.AsFloat64(price)
			}
		}
		if !cached {
			return 0
		}
		cached = false
	}
}

func (transaction *Transaction) Amount(client *Client) float64 {
	cached := true
	for {
		markets, _ := GetMarkets(client, cached)
		for i := range markets {
			price := (*transaction)[fmt.Sprintf("%s_%s", markets[i].Base, markets[i].Quote)]
			if any.AsString(price) != "" {
				return math.Abs(any.AsFloat64((*transaction)[markets[i].Base]))
			}
		}
		if !cached {
			return 0
		}
		cached = false
	}
}

func (transaction *Transaction) Side(client *Client) (string, error) {
	cached := true
	for {
		markets, _ := GetMarkets(client, cached)
		for i := range markets {
			price := (*transaction)[fmt.Sprintf("%s_%s", markets[i].Base, markets[i].Quote)]
			if any.AsString(price) != "" {
				quote := any.AsFloat64((*transaction)[markets[i].Quote])
				if quote < 0 {
					return BUY, nil
				}
				if quote > 0 {
					return SELL, nil
				}
				base := any.AsFloat64((*transaction)[markets[i].Base])
				if base < 0 {
					return SELL, nil
				}
				if base > 0 {
					return BUY, nil
				}
			}
		}
		if !cached {
			return "", errors.Errorf("unknown transaction side: %+v", transaction)
		}
		cached = false
	}
}

func (transaction *Transaction) DateTime() time.Time {
	dt := (*transaction)["datetime"]
	var out time.Time
	if dt != nil {
		out, _ = time.Parse(TimeFormat, any.AsString(dt))
	}
	return out
}

type (
	Transactions []Transaction
)

func (transactions Transactions) GetOrders() Transactions {
	var out Transactions
	for _, t := range transactions {
		if t.OrderId() != "" {
			out = append(out, t)
		}
	}
	return out
}

func (transactions Transactions) GetOrdersEx(client *Client, market string) Transactions {
	var out Transactions
	for _, t := range transactions {
		if t.OrderId() != "" {
			if market == "" || t.Market(client) == market {
				out = append(out, t)
			}
		}
	}
	return out
}

func (transactions Transactions) IndexByOrderId(id string) int {
	for i, t := range transactions {
		if t.OrderId() == id {
			return i
		}
	}
	return -1
}
