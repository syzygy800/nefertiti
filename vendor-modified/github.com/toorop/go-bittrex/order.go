package bittrex

import (
	"fmt"
	"strings"
)

const (
	OrderTypeLimit  = "LIMIT"
	OrderTypeMarket = "MARKET"
)

const (
	OrderSideBuy  = "BUY"
	OrderSideSell = "SELL"
)

const (
	// Lasts until the order is completed, expired, or cancelled. The maximum lifetime of any order is 28 days.
	// Any order older then 28 days will be automatically canceled by the system and all reserved funds will be returned to your account.
	GTC = "GOOD_TIL_CANCELLED"
	// Must be executed immediately. Any portion of an IOC order that cannot be filled immediately will be cancelled.
	IOC = "IMMEDIATE_OR_CANCEL"
	// This option allows orders to be placed which will be filled immediately and completely, or not at all.
	FOK = "FILL_OR_KILL"
)

type Order struct {
	Id            string  `json:"id"`
	MarketSymbol  string  `json:"marketSymbol"`
	Direction     string  `json:"direction"`
	OrderType     string  `json:"type"`
	Quantity      float64 `json:"quantity,string"`
	Limit         float64 `json:"limit,string"`
	Ceiling       float64 `json:"ceiling,string"`
	TimeInForce   string  `json:"timeInForce"`
	ClientOrderId string  `json:"clientOrderId"`
	FillQuantity  float64 `json:"fillQuantity,string"`
	Commission    float64 `json:"commission,string"`
	Proceeds      float64 `json:"proceeds,string"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
	ClosedAt      string  `json:"closedAt"`
}

// MarketName returns the old (v1) market name that was reversed.
func (order *Order) MarketName() string {
	symbols := strings.Split(order.MarketSymbol, "-")
	return fmt.Sprintf("%s-%s", symbols[1], symbols[0])
}

func (order *Order) QuantityFilled() float64 {
	return order.FillQuantity
}

func (order *Order) Price() float64 {
	if order.Limit > 0 {
		return order.Limit
	} else {
		return order.Proceeds / order.QuantityFilled()
	}
}

type (
	Orders []Order
)

func (orders Orders) IndexByOrderId(orderId string) int {
	for i, order := range orders {
		if order.Id == orderId {
			return i
		}
	}
	return -1
}

func (orders Orders) IndexByOrderIdEx(orderId string, side string) int {
	for i, order := range orders {
		if order.Id == orderId && order.Direction == side {
			return i
		}
	}
	return -1
}
