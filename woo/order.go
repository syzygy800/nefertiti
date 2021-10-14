package woo

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type (
	OrderSide   string
	OrderType   string
	OrderStatus string
)

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

const (
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeMarket OrderType = "MARKET"
)

const (
	OrderStatusNew           OrderStatus = "NEW"
	OrderStatusCancelled     OrderStatus = "CANCELLED"
	OrderStatusPartialFilled OrderStatus = "PARTIAL_FILLED"
	OrderStatusFilled        OrderStatus = "FILLED"
	OrderStatusRejected      OrderStatus = "REJECTED"
	OrderStatusIncomplete    OrderStatus = "INCOMPLETE"
	OrderStatusCompleted     OrderStatus = "COMPLETED"
)

type NewOrder struct {
	ID       int64       `json:"order_id"`
	Type     string      `json:"order_type"`
	Price    float64     `json:"order_price"`
	Quantity float64     `json:"order_quantity"`
	Amount   interface{} `json:"order_amount"`
}

func (client *Client) Order(symbol string, side OrderSide, orderType OrderType, quantity, price float64) (*NewOrder, error) {
	params := url.Values{}
	params.Add("symbol", symbol)
	params.Add("order_type", string(orderType))
	params.Add("order_quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	params.Add("side", string(side))
	if orderType == OrderTypeLimit {
		params.Add("order_price", strconv.FormatFloat(price, 'f', -1, 64))
	}

	var (
		err  error
		body []byte
		out  NewOrder
	)
	if body, err = client.call(http.MethodPost, "/v1/order", params, 2); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(body, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (client *Client) CancelOrder(symbol string, orderID int64) error {
	params := url.Values{}
	params.Add("symbol", symbol)
	params.Add("order_id", strconv.FormatInt(orderID, 10))

	_, err := client.call(http.MethodDelete, "/v1/order", params, 20)

	return err
}

type Order struct {
	Symbol        string      `json:"symbol"`
	Status        OrderStatus `json:"status"`
	Side          OrderSide   `json:"side"`
	CreatedTime   string      `json:"created_time"`
	UpdatedTime   string      `json:"updated_time"`
	OrderID       int64       `json:"order_id"`
	ApplicationID string      `json:"application_id"`
	OrderTag      string      `json:"order_tag"`
	Price         float64     `json:"price"`
	Type          OrderType   `json:"type"`
	Quantity      float64     `json:"quantity"`
	Amount        interface{} `json:"amount"`
}

func (order *Order) CreatedAt() time.Time {
	sec, err := strconv.ParseInt(order.CreatedTime, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(sec, 0)
}

func (order *Order) UpdatedAt() time.Time {
	sec, err := strconv.ParseInt(order.UpdatedTime, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(sec, 0)
}

type Orders struct {
	Meta struct {
		RecordsPerPage int64 `json:"records_per_page"`
		CurrentPage    int64 `json:"current_page"`
	} `json:"meta"`
	Rows []Order `json:"rows"`
}

func (client *Client) Orders(symbol string, status OrderStatus) ([]Order, error) {
	var (
		page   int64 = 1
		result []Order
	)

	for {
		params := url.Values{}
		params.Add("symbol", symbol)
		params.Add("status", string(status))
		params.Add("page", strconv.FormatInt(page, 10))

		var (
			err  error
			body []byte
			out  *Orders
		)
		if body, err = client.get("/v1/orders", params, true, 10); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(body, &out); err != nil {
			return nil, err
		}

		result = append(result, out.Rows...)

		if len(out.Rows) == 0 || len(out.Rows) < int(out.Meta.RecordsPerPage) {
			break
		}

		page++
	}

	return result, nil
}
