package binance

import (
	"net/http"
)

type Request int

const (
	ALL_ORDERS Request = iota
	CANCEL_ORDER
	CREATE_OCO_ORDER
	CREATE_ORDER
	DEPT
	EXCHANGE_INFO
	OPEN_ORDERS_WITH_SYMBOL
	OPEN_ORDERS_WITHOUT_SYMBOL
	TICKER_24H_WITH_SYMBOL
	TICKER_24H_WITHOUT_SYMBOL
)

var Weight = map[Request]int{
	ALL_ORDERS:                 10,
	CANCEL_ORDER:               1,
	CREATE_OCO_ORDER:           1,
	CREATE_ORDER:               1,
	DEPT:                       1,
	EXCHANGE_INFO:              10,
	OPEN_ORDERS_WITH_SYMBOL:    3,
	OPEN_ORDERS_WITHOUT_SYMBOL: 40,
	TICKER_24H_WITH_SYMBOL:     1,
	TICKER_24H_WITHOUT_SYMBOL:  40,
}

var Method = map[Request]string{
	ALL_ORDERS:                 http.MethodGet,
	CANCEL_ORDER:               http.MethodDelete,
	CREATE_OCO_ORDER:           http.MethodPost,
	CREATE_ORDER:               http.MethodPost,
	DEPT:                       http.MethodGet,
	EXCHANGE_INFO:              http.MethodGet,
	OPEN_ORDERS_WITH_SYMBOL:    http.MethodGet,
	OPEN_ORDERS_WITHOUT_SYMBOL: http.MethodGet,
	TICKER_24H_WITH_SYMBOL:     http.MethodGet,
	TICKER_24H_WITHOUT_SYMBOL:  http.MethodGet,
}

var Path = map[Request]string{
	ALL_ORDERS:                 "/api/v3/allOrders?symbol=%s",
	CANCEL_ORDER:               "/api/v3/order?symbol=%s",
	CREATE_OCO_ORDER:           "/api/v3/order/oco?symbol=%s&side=%s&quantity=%f&price=%f&stopPrice=%f",
	CREATE_ORDER:               "/api/v3/order?symbol=%s&side=%s&type=%s",
	DEPT:                       "/api/v3/depth?symbol=%s",
	EXCHANGE_INFO:              "/api/v3/exchangeInfo",
	OPEN_ORDERS_WITH_SYMBOL:    "/api/v3/openOrders?symbol=%s",
	OPEN_ORDERS_WITHOUT_SYMBOL: "/api/v3/openOrders",
	TICKER_24H_WITH_SYMBOL:     "/api/v3/ticker/24hr?symbol=%s",
	TICKER_24H_WITHOUT_SYMBOL:  "/api/v3/ticker/24hr",
}
