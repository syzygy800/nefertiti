//lint:file-ignore ST1006 receiver name should be a reflection of its identity; don't use generic names such as "this" or "self"
package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	exchange "github.com/adshao/go-binance/v2"
)

//---------------------------- CreateOrderService -----------------------------

type CreateOrderService struct {
	client    *Client
	symbol    string
	side      exchange.SideType
	orderType exchange.OrderType
	inner     *exchange.CreateOrderService
}

func (self *CreateOrderService) Symbol(symbol string) *CreateOrderService {
	self.symbol = symbol
	self.inner.Symbol(symbol)
	return self
}

func (self *CreateOrderService) Quantity(quantity float64) *CreateOrderService {
	self.inner.Quantity(strconv.FormatFloat(quantity, 'f', -1, 64))
	return self
}

func (self *CreateOrderService) NewClientOrderID(newClientOrderID string) *CreateOrderService {
	self.inner.NewClientOrderID(newClientOrderID)
	return self
}

func (self *CreateOrderService) Type(orderType exchange.OrderType) *CreateOrderService {
	self.orderType = orderType
	self.inner.Type(orderType)
	return self
}

func (self *CreateOrderService) TimeInForce(timeInForce exchange.TimeInForceType) *CreateOrderService {
	self.inner.TimeInForce(timeInForce)
	return self
}

func (self *CreateOrderService) Price(price float64) *CreateOrderService {
	self.inner.Price(strconv.FormatFloat(price, 'f', -1, 64))
	return self
}

func (self *CreateOrderService) Side(side exchange.SideType) *CreateOrderService {
	self.side = side
	self.inner.Side(side)
	return self
}

func (self *CreateOrderService) StopPrice(stopPrice float64) *CreateOrderService {
	self.inner.StopPrice(strconv.FormatFloat(stopPrice, 'f', -1, 64))
	return self
}

func (self *CreateOrderService) Do(ctx context.Context, opts ...exchange.RequestOption) (*exchange.CreateOrderResponse, error) {
	defer AfterRequest()
	BeforeRequest(self.client, Method[CREATE_ORDER], fmt.Sprintf(Path[CREATE_ORDER], self.symbol, self.side, self.orderType), Weight[CREATE_ORDER])
	res, err := self.inner.Do(ctx, opts...)
	self.client.handleError(err)
	return res, err
}

//---------------------------- NewCreateOCOService ----------------------------

type CreateOCOService struct {
	client    *Client
	symbol    string
	side      exchange.SideType
	quantity  float64
	price     float64
	stopPrice float64
	inner     *exchange.CreateOCOService
}

func (self *CreateOCOService) Symbol(symbol string) *CreateOCOService {
	self.symbol = symbol
	self.inner.Symbol(symbol)
	return self
}

func (self *CreateOCOService) Side(side exchange.SideType) *CreateOCOService {
	self.side = side
	self.inner.Side(side)
	return self
}

func (self *CreateOCOService) Quantity(quantity float64) *CreateOCOService {
	self.quantity = quantity
	self.inner.Quantity(strconv.FormatFloat(quantity, 'f', -1, 64))
	return self
}

func (self *CreateOCOService) Price(price float64) *CreateOCOService {
	self.price = price
	self.inner.Price(strconv.FormatFloat(price, 'f', -1, 64))
	return self
}

func (self *CreateOCOService) StopPrice(stopPrice float64) *CreateOCOService {
	self.stopPrice = stopPrice
	self.inner.StopPrice(strconv.FormatFloat(stopPrice, 'f', -1, 64))
	return self
}

func (self *CreateOCOService) StopClientOrderID(stopClientOrderID string) *CreateOCOService {
	self.inner.StopClientOrderID(stopClientOrderID)
	return self
}

func (self *CreateOCOService) LimitClientOrderID(limitClientOrderID string) *CreateOCOService {
	self.inner.LimitClientOrderID(limitClientOrderID)
	return self
}

func (self *CreateOCOService) StopLimitPrice(stopLimitPrice float64) *CreateOCOService {
	self.inner.StopLimitPrice(strconv.FormatFloat(stopLimitPrice, 'f', -1, 64))
	return self
}

func (self *CreateOCOService) StopLimitTimeInForce(stopLimitTimeInForce exchange.TimeInForceType) *CreateOCOService {
	self.inner.StopLimitTimeInForce(stopLimitTimeInForce)
	return self
}

func (self *CreateOCOService) Do(ctx context.Context, opts ...exchange.RequestOption) (*exchange.CreateOCOResponse, error) {
	defer AfterRequest()
	BeforeRequest(self.client, Method[CREATE_OCO_ORDER], fmt.Sprintf(Path[CREATE_OCO_ORDER], self.symbol, self.side, self.quantity, self.price, self.stopPrice), Weight[CREATE_OCO_ORDER])
	res, err := self.inner.Do(ctx, opts...)
	self.client.handleError(err)
	return res, err
}

//----------------------------------- Order -----------------------------------

type Order struct {
	*exchange.Order
}

// GetPrice returns the Price as float64
func (self *Order) GetPrice() float64 {
	var (
		err error
		out float64
	)

	// if a STOP_LOSS_LIMIT order got filled, assume the stopPrice is the price things got executed at
	if self.Type == exchange.OrderTypeStopLossLimit {
		out, err = strconv.ParseFloat(self.StopPrice, 64)
		if err == nil && out > 0 {
			return out
		}
	}

	out, err = strconv.ParseFloat(self.Price, 64)
	if err == nil {
		// if we don't have a price, divide cummulativeQuoteQty by executedQty
		if out == 0 {
			var (
				cq float64 // cummulativeQuoteQty
				eq float64 // executedQty
			)
			cq, err = strconv.ParseFloat(self.CummulativeQuoteQuantity, 64)
			if err == nil && cq > 0 {
				eq, err = strconv.ParseFloat(self.ExecutedQuantity, 64)
				if err == nil && eq > 0 {
					return cq / eq
				}
			}
		}
		return out
	}

	return 0
}

// GetSize returns the Quantity as float64
func (self *Order) GetSize() float64 {
	out, err := strconv.ParseFloat(self.OrigQuantity, 64)
	if err == nil {
		return out
	}
	return 0
}

// GetStopPrice returns the StopPrice as float64
func (self *Order) GetStopPrice() float64 {
	out, err := strconv.ParseFloat(self.StopPrice, 64)
	if err == nil {
		return out
	}
	return 0
}

// CreatedAt returns the order creation time as time.Time
func (self *Order) CreatedAt() time.Time {
	if self.Time == 0 {
		return time.Now()
	} else {
		return time.Unix((self.Time / 1000), 0)
	}
}

// UpdatedAt returns the last time the order was updated as time.Time
func (self *Order) UpdatedAt() time.Time {
	if self.UpdateTime == 0 {
		return time.Now()
	} else {
		return time.Unix((self.UpdateTime / 1000), 0)
	}
}

func wrap(input *exchange.Order) (*Order, error) {
	var (
		err error
		buf []byte
		out Order
	)
	if buf, err = json.Marshal(input); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(buf, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
