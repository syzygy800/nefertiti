//lint:file-ignore ST1006 receiver name should be a reflection of its identity; don't use generic names such as "this" or "self"
package binance

import (
	"context"
	"net/http"
	"time"

	exchange "github.com/adshao/go-binance/v2"
)

const (
	BASE_URL    = "https://api.binance.com"
	BASE_URL_1  = "https://api1.binance.com"
	BASE_URL_2  = "https://api2.binance.com"
	BASE_URL_3  = "https://api3.binance.com"
	BASE_URL_US = "https://api.binance.us"
)

type Client struct {
	inner *exchange.Client
}

// Get all account orders; active, canceled, or filled.
func (self *Client) Orders(symbol string) ([]Order, error) {
	var (
		err    error
		orders []*exchange.Order
		output []Order
	)
	defer AfterRequest()
	BeforeRequest(self, WEIGHT_ALL_ORDERS)
	if orders, err = self.inner.NewListOrdersService().Symbol(symbol).Do(context.Background()); err != nil {
		self.handleError(err)
		return nil, err
	}
	for _, unwrapped := range orders {
		var wrapped *Order
		if wrapped, err = wrap(unwrapped); err != nil {
			return nil, err
		}
		output = append(output, *wrapped)
	}
	return output, nil
}

// Get all open orders without a symbol.
func (self *Client) OpenOrders() ([]Order, error) {
	var (
		err    error
		orders []*exchange.Order
		output []Order
	)
	defer AfterRequest()
	BeforeRequest(self, WEIGHT_OPEN_ORDERS_WITHOUT_SYMBOL)
	if orders, err = self.inner.NewListOpenOrdersService().Do(context.Background()); err != nil {
		self.handleError(err)
		return nil, err
	}
	for _, unwrapped := range orders {
		var wrapped *Order
		if wrapped, err = wrap(unwrapped); err != nil {
			return nil, err
		}
		output = append(output, *wrapped)
	}
	return output, nil
}

// Get all open orders on a symbol.
func (self *Client) OpenOrdersEx(symbol string) ([]Order, error) {
	var (
		err    error
		orders []*exchange.Order
		output []Order
	)
	defer AfterRequest()
	BeforeRequest(self, WEIGHT_OPEN_ORDERS_WITH_SYMBOL)
	if orders, err = self.inner.NewListOpenOrdersService().Symbol(symbol).Do(context.Background()); err != nil {
		self.handleError(err)
		return nil, err
	}
	for _, unwrapped := range orders {
		var wrapped *Order
		if wrapped, err = wrap(unwrapped); err != nil {
			return nil, err
		}
		output = append(output, *wrapped)
	}
	return output, nil
}

// Cancel an active order.
func (self *Client) CancelOrder(symbol string, orderID int64) error {
	defer AfterRequest()
	BeforeRequest(self, WEIGHT_CANCEL_ORDER)
	_, err := self.inner.NewCancelOrderService().Symbol(symbol).OrderID(orderID).Do(context.Background())
	self.handleError(err)
	return err
}

func (self *Client) NewCreateOrderService() *CreateOrderService {
	return &CreateOrderService{client: self, inner: self.inner.NewCreateOrderService()}
}

func (self *Client) NewCreateOCOService() *CreateOCOService {
	return &CreateOCOService{client: self, inner: self.inner.NewCreateOCOService()}
}

func (self *Client) handleError(err error) {
	if err == nil {
		return
	}
	binanceError, ok := IsBinanceError(err)
	if ok {
		if binanceError.Code == -1021 {
			// Timestamp for this request is outside of the recvWindow.
			if offset, err := self.inner.NewSetServerTimeService().Do(context.Background()); err == nil {
				SERVER_TIME_OFFSET = offset
				SERVER_TIME_UPDATE = time.Now()
			}
		}
	}
}

var (
	SERVER_TIME_OFFSET int64     // offset between device time and server time
	SERVER_TIME_UPDATE time.Time // the last time we asked for server time
)

func New(baseURL, apiKey, apiSecret string) *Client {
	client := exchange.NewClient(apiKey, apiSecret)

	client.BaseURL = baseURL
	client.HTTPClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	if SERVER_TIME_OFFSET == 0 || time.Since(SERVER_TIME_UPDATE).Minutes() > 15 {
		if offset, err := client.NewSetServerTimeService().Do(context.Background()); err == nil {
			SERVER_TIME_OFFSET = offset
			SERVER_TIME_UPDATE = time.Now()
		}
	} else {
		client.TimeOffset = SERVER_TIME_OFFSET
	}

	return &Client{inner: client}
}
