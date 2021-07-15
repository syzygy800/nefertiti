//lint:file-ignore ST1006 receiver name should be a reflection of its identity; don't use generic names such as "this" or "self"
package bittrex

import (
	"fmt"
	"strconv"
	"strings"
)

type OrderId string

//----------------------- OrderType -----------------------

type OrderType int

const (
	LIMIT OrderType = iota
	MARKET
)

var OrderTypeString = map[OrderType]string{
	LIMIT:  "LIMIT",
	MARKET: "MARKET",
}

func (ot *OrderType) String() string {
	return OrderTypeString[*ot]
}

//----------------------- OrderSide -----------------------

type OrderSide int

const (
	BUY OrderSide = iota
	SELL
)

var OrderSideString = map[OrderSide]string{
	BUY:  "BUY",
	SELL: "SELL",
}

func (os *OrderSide) String() string {
	return OrderSideString[*os]
}

//---------------------- TimeInForce ----------------------

type TimeInForce int

const (
	// Lasts until the order is completed, expired, or cancelled. The maximum lifetime of any order is 28 days.
	// Any order older then 28 days will be automatically canceled by the system and all reserved funds will be returned to your account.
	GTC TimeInForce = iota
	// Must be executed immediately. Any portion of an IOC order that cannot be filled immediately will be cancelled.
	IOC
	// This option allows orders to be placed which will be filled immediately and completely, or not at all.
	FOK
)

var TimeInForceString = map[TimeInForce]string{
	GTC: "GOOD_TIL_CANCELLED",
	IOC: "IMMEDIATE_OR_CANCEL",
	FOK: "FILL_OR_KILL",
}

func (tif *TimeInForce) String() string {
	return TimeInForceString[*tif]
}

//------------------------ Operand ------------------------

type Operand int

const (
	LTE Operand = iota
	GTE
)

var OperandString = map[Operand]string{
	LTE: "LTE",
	GTE: "GTE",
}

func (op *Operand) String() string {
	return OperandString[*op]
}

//----------------------- NewOrder ------------------------

type newOrder struct {
	MarketSymbol string `json:"marketSymbol"`
	Direction    string `json:"direction"` // BUY or SELL
	OrderType    string `json:"type"`      // LIMIT or MARKET
	Quantity     string `json:"quantity"`
	Limit        string `json:"limit,omitempty"`
	TimeInForce  string `json:"timeInForce"`
}

type NewOrder struct {
	MarketSymbol string
	Direction    OrderSide
	OrderType    OrderType
	Quantity     float64
	Limit        float64
	TimeInForce  TimeInForce
}

func (self *NewOrder) into() newOrder {
	return newOrder{
		MarketSymbol: self.MarketSymbol,
		Direction:    self.Direction.String(),
		OrderType:    self.OrderType.String(),
		Quantity:     strconv.FormatFloat(self.Quantity, 'f', -1, 64),
		TimeInForce:  self.TimeInForce.String(),
	}
}

//------------------------- Order -------------------------

type Order struct {
	Id            OrderId                    `json:"id"`
	MarketSymbol  string                     `json:"marketSymbol"`
	Direction     string                     `json:"direction"` // BUY or SELL
	OrderType     string                     `json:"type"`      // LIMIT or MARKET
	Quantity      float64                    `json:"quantity,string"`
	Limit         float64                    `json:"limit,string,omitempty"`
	Ceiling       float64                    `json:"ceiling,string,omitempty"`
	TimeInForce   string                     `json:"timeInForce,omitempty"`
	ClientOrderId OrderId                    `json:"clientOrderId,omitempty"`
	FillQuantity  float64                    `json:"fillQuantity,string,omitempty"`
	Commission    float64                    `json:"commission,string,omitempty"`
	Proceeds      float64                    `json:"proceeds,string,omitempty"`
	Status        string                     `json:"status,omitempty"` // OPEN or CLOSED
	CreatedAt     string                     `json:"createdAt,omitempty"`
	UpdatedAt     string                     `json:"updatedAt,omitempty"`
	ClosedAt      string                     `json:"closedAt,omitempty"`
	OrderToCancel *newCancelConditionalOrder `json:"orderToCancel,omitempty"`
}

// MarketName returns the old (v1) market name that was reversed.
func (order *Order) MarketName() string {
	symbols := strings.Split(order.MarketSymbol, "-")
	return fmt.Sprintf("%s-%s", symbols[1], symbols[0])
}

func (order *Order) Type() OrderType {
	if order.OrderType == OrderTypeString[MARKET] {
		return MARKET
	} else {
		return LIMIT
	}
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

//------------------------ Orders -------------------------

type (
	Orders []Order
)

func (orders Orders) IndexByOrderId(id OrderId) int {
	for i, order := range orders {
		if order.Id == id {
			return i
		}
	}
	return -1
}

func (orders Orders) IndexByOrderIdEx(id OrderId, side OrderSide) int {
	for i, order := range orders {
		if order.Id == id && order.Direction == OrderSideString[side] {
			return i
		}
	}
	return -1
}

//------------------- ConditionalOrder --------------------

type newCancelConditionalOrder struct {
	OrderType string  `json:"type"` // ORDER or CONDITIONAL_ORDER
	Id        OrderId `json:"id"`
}

type ConditionalOrder struct {
	Id                       OrderId                    `json:"id"`
	MarketSymbol             string                     `json:"marketSymbol"`
	Operand                  string                     `json:"operand"` // LTE or GTE
	TriggerPrice             float64                    `json:"triggerPrice,string"`
	TrailingStopPercent      float64                    `json:"trailingStopPercent,string"`
	CreatedOrderId           OrderId                    `json:"createdOrderId"`
	OrderToCreate            *newOrder                  `json:"orderToCreate"`
	OrderToCancel            *newCancelConditionalOrder `json:"orderToCancel"`
	ClientConditionalOrderId OrderId                    `json:"clientConditionalOrderId"`
	Status                   string                     `json:"status"` // OPEN or COMPLETED or CANCELLED or FAILED
	OrderCreationErrorCode   string                     `json:"orderCreationErrorCode"`
	CreatedAt                string                     `json:"createdAt"`
	UpdatedAt                string                     `json:"updatedAt"`
	ClosedAt                 string                     `json:"closedAt"`
}
