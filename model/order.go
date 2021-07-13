package model

import (
	"encoding/json"
	"time"
)

type OrderSide int

const (
	ORDER_SIDE_NONE OrderSide = iota
	BUY
	SELL
)

var OrderSideString = map[OrderSide]string{
	ORDER_SIDE_NONE: "",
	BUY:             "buy",
	SELL:            "sell",
}

func (os *OrderSide) String() string {
	return OrderSideString[*os]
}

func NewOrderSide(data string) OrderSide {
	for os := range OrderSideString {
		if os.String() == data {
			return os
		}
	}
	return ORDER_SIDE_NONE
}

func FormatOrderSide(value OrderSide) string {
	if value == BUY {
		return "Buy"
	} else {
		if value == SELL {
			return "Sell"
		}
	}
	return ""
}

type OrderType int

const (
	ORDER_TYPE_NONE OrderType = iota
	LIMIT
	MARKET
)

var OrderTypeString = map[OrderType]string{
	ORDER_TYPE_NONE: "",
	LIMIT:           "limit",
	MARKET:          "market",
}

func (ot *OrderType) String() string {
	return OrderTypeString[*ot]
}

func NewOrderType(data string) OrderType {
	for ot := range OrderTypeString {
		if ot.String() == data {
			return ot
		}
	}
	return ORDER_TYPE_NONE
}

type (
	Order struct {
		Side      OrderSide `json:"-"`
		Market    string    `json:"market"`
		Size      float64   `json:"size"`
		Price     float64   `json:"price"`
		CreatedAt time.Time `json:"createdAt"`
	}
	Orders []Order
)

func (order *Order) MarshalJSON() ([]byte, error) {
	type Alias Order
	return json.Marshal(&struct {
		Side string `json:"side"`
		*Alias
	}{
		Side:  order.Side.String(),
		Alias: (*Alias)(order),
	})
}

func (orders Orders) IndexByPrice(side OrderSide, market string, price float64) int {
	for i, order := range orders {
		if order.Side == side && order.Market == market && order.Price == price {
			return i
		}
	}
	return -1
}

func (orders Orders) OrderByPrice(side OrderSide, market string, price float64) *Order {
	i := orders.IndexByPrice(side, market, price)
	if i != -1 {
		return &orders[i]
	}
	return nil
}

func (orders Orders) Youngest(side OrderSide, def time.Time) time.Time {
	youngest := time.Time{} // January 1, year 1, 00:00:00.000000000 UTC
	for _, order := range orders {
		if order.Side == side {
			if youngest.IsZero() || youngest.Before(order.CreatedAt) {
				youngest = order.CreatedAt
			}
		}
	}
	if youngest.IsZero() {
		return def
	} else {
		return youngest
	}
}
