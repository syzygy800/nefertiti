package kucoin

import (
	"net/http"
	"strconv"
	"time"
)

// A CreateOrderResultModel represents the result of CreateOrder().
type CreateOrderResultModel struct {
	OrderId string `json:"orderId"`
}

// CreateOrder places a new order.
func (as *ApiService) CreateOrder(params map[string]string) (*ApiResponse, error) {
	req := NewRequest(http.MethodPost, "/api/v1/orders", params)
	return as.Call(req)
}

// CreateStopOrder places a new stop-order.
func (as *ApiService) CreateStopOrder(params map[string]string) (*ApiResponse, error) {
	req := NewRequest(http.MethodPost, "/api/v1/stop-order", params)
	return as.Call(req)
}

// A CancelOrderResultModel represents the result of CancelOrder().
type CancelOrderResultModel struct {
	CancelledOrderIds []string `json:"cancelledOrderIds"`
}

// CancelOrder cancels a previously placed order.
func (as *ApiService) CancelOrder(orderId string) (*ApiResponse, error) {
	req := NewRequest(http.MethodDelete, "/api/v1/orders/"+orderId, nil)
	return as.Call(req)
}

// CancelStopOrder cancels a previously placed stop-order.
func (as *ApiService) CancelStopOrder(orderId string) (*ApiResponse, error) {
	req := NewRequest(http.MethodDelete, "/api/v1/stop-order/"+orderId, nil)
	return as.Call(req)
}

// An OrderModel represents an order.
type OrderModel struct {
	Id            string `json:"id"`
	Symbol        string `json:"symbol"`
	OpType        string `json:"opType"`
	Type          string `json:"type"`
	Side          string `json:"side"`
	Price         string `json:"price"`
	Size          string `json:"size"`
	Funds         string `json:"funds,omitempty"`
	DealFunds     string `json:"dealFunds,omitempty"`
	DealSize      string `json:"dealSize,omitempty"`
	Fee           string `json:"fee,omitempty"`
	FeeCurrency   string `json:"feeCurrency,omitempty"`
	Stp           string `json:"stp,omitempty"`
	Stop          string `json:"stop,omitempty"`
	StopTriggered bool   `json:"stopTriggered,omitempty"`
	StopPrice     string `json:"stopPrice,omitempty"`
	TimeInForce   string `json:"timeInForce,omitempty"`
	PostOnly      bool   `json:"postOnly,omitempty"`
	Hidden        bool   `json:"hidden,omitempty"`
	IceBerg       bool   `json:"iceberg,omitempty"`
	VisibleSize   string `json:"visibleSize,omitempty"`
	CancelAfter   uint64 `json:"cancelAfter,omitempty"`
	Channel       string `json:"channel,omitempty"`
	ClientOid     string `json:"clientOid,omitempty"`
	Remark        string `json:"remark,omitempty"`
	Tags          string `json:"tags,omitempty"`
	IsActive      bool   `json:"isActive,omitempty"`
	CancelExist   bool   `json:"cancelExist,omitempty"`
	CreatedAt     int64  `json:"createdAt,omitempty"`
	TradeType     string `json:"tradeType,omitempty"`
	Status        string `json:"status,omitempty"`
	FailMsg       string `json:"failMsg,omitempty"`
}

// ParseStopPrice returns the StopPrice as float64
func (o *OrderModel) ParseStopPrice() float64 {
	out, err := strconv.ParseFloat(o.StopPrice, 64)
	if err == nil {
		return out
	}
	return 0
}

// ParsePrice returns the Price as float64
func (o *OrderModel) ParsePrice() float64 {
	out, err := strconv.ParseFloat(o.Price, 64)
	if err == nil {
		return out
	}
	return 0
}

// ParseSize returns the Quantity as float64
func (o *OrderModel) ParseSize() float64 {
	out, err := strconv.ParseFloat(o.Size, 64)
	if err == nil {
		return out
	}
	return 0
}

// ParseCreatedAt returns the creation time as time.Time
func (o *OrderModel) ParseCreatedAt() time.Time {
	return time.Unix(o.CreatedAt/1000, 0)
}

// A OrdersModel is the set of *OrderModel.
type (
	OrdersModel    []*OrderModel
	OrderPredicate func(order *OrderModel) bool
)

func (orders OrdersModel) Find(callback *OrderPredicate) int {
	cb := *callback
	for idx, order := range orders {
		if cb(order) {
			return idx
		}
	}
	return -1
}

func (orders OrdersModel) IndexOfId(Id string) int {
	var cb OrderPredicate = func(order *OrderModel) bool {
		return order.Id == Id
	}
	return orders.Find(&cb)
}

// Orders returns a list your current orders.
func (as *ApiService) Orders(params map[string]string, pagination *PaginationParam) (*ApiResponse, error) {
	pagination.ReadParam(params)
	req := NewRequest(http.MethodGet, "/api/v1/orders", params)
	return as.Call(req)
}

// StopOrders returns a list your current stop-orders.
func (as *ApiService) StopOrders(params map[string]string, pagination *PaginationParam) (*ApiResponse, error) {
	pagination.ReadParam(params)
	req := NewRequest(http.MethodGet, "/api/v1/stop-order", params)
	return as.Call(req)
}

// Order returns a single order by order id.
func (as *ApiService) Order(orderId string) (*ApiResponse, error) {
	req := NewRequest(http.MethodGet, "/api/v1/orders/"+orderId, nil)
	return as.Call(req)
}

// StopOrder returns a single order by stop-order id.
func (as *ApiService) StopOrder(orderId string) (*ApiResponse, error) {
	req := NewRequest(http.MethodGet, "/api/v1/stop-order/"+orderId, nil)
	return as.Call(req)
}

// RecentOrders returns the recent orders of the latest transactions within 24 hours.
func (as *ApiService) RecentOrders() (*ApiResponse, error) {
	req := NewRequest(http.MethodGet, "/api/v1/limit/orders", nil)
	return as.Call(req)
}
