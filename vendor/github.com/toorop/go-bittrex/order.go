package bittrex

import (
	"fmt"
	"strings"
)

const (
	OrderTypeLimitBuy   = "LIMIT_BUY"
	OrderTypeMarketBuy  = "MARKET_BUY"
	OrderTypeLimitSell  = "LIMIT_SELL"
	OrderTypeMarketSell = "MARKET_SELL"
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

const (
	ConditionNone        = "NONE"
	LessThanOrEqualTo    = "LESS_THAN"
	GreaterThanOrEqualTo = "GREATER_THAN"
)

type Order struct {
	OrderUuid         string  `json:"OrderUuid"`
	Exchange          string  `json:"Exchange"`
	OrderType         string  `json:"OrderType"`
	Quantity          float64 `json:"Quantity"`
	QuantityRemaining float64 `json:"QuantityRemaining"`
	Limit             float64 `json:"Limit"`
	CommissionPaid    float64 `json:"CommissionPaid"`
	Price             float64 `json:"Price"`
	PricePerUnit      float64 `json:"PricePerUnit"`
	Opened            string  `json:"Opened"`
	Closed            string  `json:"Closed"`
	CancelInitiated   bool    `json:"CancelInitiated"`
	ImmediateOrCancel bool    `json:"ImmediateOrCancel"`
	IsConditional     bool    `json:"IsConditional"`
	Condition         string  `json:"Condition"`
	ConditionTarget   float64 `json:"ConditionTarget"`
}

type (
	Orders []Order
)

func (orders Orders) IndexByUuid(uuid string) int {
	for i, order := range orders {
		if order.OrderUuid == uuid {
			return i
		}
	}
	return -1
}

type Fill struct {
	OrderUuid         string  `json:"OrderUuid"`
	Exchange          string  `json:"Exchange"`
	TimeStamp         string  `json:"TimeStamp"`
	OrderType         string  `json:"OrderType"`
	Limit             float64 `json:"Limit"`
	Quantity          float64 `json:"Quantity"`
	QuantityRemaining float64 `json:"QuantityRemaining"`
	Commission        float64 `json:"Commission"`
	Price             float64 `json:"Price"`
	PricePerUnit      float64 `json:"PricePerUnit"`
	IsConditional     bool    `json:"IsConditional"`
	Condition         string  `json:"Condition"`
	ConditionTarget   float64 `json:"ConditionTarget"`
	ImmediateOrCancel bool    `json:"ImmediateOrCancel"`
	Closed            string  `json:"Closed"`
}

func (fill *Fill) QuantityFilled() float64 {
	return (fill.Quantity - fill.QuantityRemaining)
}

func (fill *Fill) BoughtAt() float64 {
	if fill.Limit > 0 {
		return fill.Limit
	} else {
		return fill.Price / fill.QuantityFilled()
	}
}

type (
	Fills []Fill
)

func (fills Fills) IndexByUuid(uuid string) int {
	for i, fill := range fills {
		if fill.OrderUuid == uuid {
			return i
		}
	}
	return -1
}

func (fills Fills) IndexByUuidEx(uuid string, side string) int {
	for i, fill := range fills {
		if fill.OrderUuid == uuid && fill.OrderType == side {
			return i
		}
	}
	return -1
}

type OrderEx struct {
	OrderUuid                  string  `json:"OrderUuid"`
	Exchange                   string  `json:"Exchange"`
	Type                       string  `json:"Type"`
	Quantity                   float64 `json:"Quantity"`
	QuantityRemaining          float64 `json:"QuantityRemaining"`
	Limit                      float64 `json:"Limit"`
	Reserved                   float64 `json:"Reserved"`
	ReserveRemaining           float64 `json:"ReserveRemaining"`
	CommissionReserved         float64 `json:"CommissionReserved"`
	CommissionReserveRemaining float64 `json:"CommissionReserveRemaining"`
	CommissionPaid             float64 `json:"CommissionPaid"`
	Price                      float64 `json:"Price"`
	PricePerUnit               float64 `json:"PricePerUnit"`
	Opened                     string  `json:"Opened"`
	Closed                     string  `json:"Closed"`
	IsOpen                     bool    `json:"IsOpen"`
	Sentinel                   string  `json:"Sentinel"`
	CancelInitiated            bool    `json:"CancelInitiated"`
	ImmediateOrCancel          bool    `json:"ImmediateOrCancel"`
	IsConditional              bool    `json:"IsConditional"`
	Condition                  string  `json:"Condition"`
	ConditionTarget            float64 `json:"ConditionTarget"`
}

type Order2 struct {
	OrderId        string  `json:"OrderId"`
	MarketName     string  `json:"MarketName"`
	MarketCurrency string  `json:"MarketCurrency"`
	BuyOrSell      string  `json:"BuyOrSell"`
	OrderType      string  `json:"OrderType"`
	Quantity       float64 `json:"Quantity"`
	Rate           float64 `json:"Rate"`
}

type Order3 struct {
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
func (order *Order3) MarketName() string {
	symbols := strings.Split(order.MarketSymbol, "-")
	return fmt.Sprintf("%s-%s", symbols[1], symbols[0])
}

func (order *Order3) QuantityFilled() float64 {
	return order.FillQuantity
}

func (order *Order3) Price() float64 {
	if order.Limit > 0 {
		return order.Limit
	} else {
		return order.Proceeds / order.QuantityFilled()
	}
}

type (
	Orders3 []Order3
)

func (orders Orders3) IndexByOrderId(orderId string) int {
	for i, order := range orders {
		if order.Id == orderId {
			return i
		}
	}
	return -1
}

func (orders Orders3) IndexByOrderIdEx(orderId string, side string) int {
	for i, order := range orders {
		if order.Id == orderId && order.Direction == side {
			return i
		}
	}
	return -1
}
