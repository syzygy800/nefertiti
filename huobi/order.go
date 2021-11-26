package huobi

import (
	"encoding/json"
	"net/url"
	"time"
)

type (
	OrderType  string
	OrderState string
)

const (
	OrderTypeBuyMarket        OrderType = "buy-market"
	OrderTypeSellMarket       OrderType = "sell-market"
	OrderTypeBuyLimit         OrderType = "buy-limit"
	OrderTypeSellLimit        OrderType = "sell-limit"
	OrderTypeBuyIOC           OrderType = "buy-ioc"
	OrderTypeSellIOC          OrderType = "sell-ioc"
	OrderTypeBuyLimitMaker    OrderType = "buy-limit-maker"
	OrderTypeSellLimitMaker   OrderType = "sell-limit-maker"
	OrderTypeBuyStopLimit     OrderType = "buy-stop-limit"
	OrderTypeSellStopLimit    OrderType = "sell-stop-limit"
	OrderTypeBuyLimitFOK      OrderType = "buy-limit-fok"
	OrderTypeSellLimitFOK     OrderType = "sell-limit-fok"
	OrderTypeBuyStopLimitFOK  OrderType = "buy-stop-limit-fok"
	OrderTypeSellStopLimitFOK OrderType = "sell-stop-limit-fok"
)

const (
	OrderStateCreated       OrderState = "created"
	OrderStateSubmitted     OrderState = "submitted"
	OrderStatePartialFilled OrderState = "partial-filled"
)

type Order struct {
	Symbol           string     `json:"symbol"` // the trading symbol to trade, e.g. btcusdt, bccbtc
	Amount           float64    `json:"amount,string"`
	Price            float64    `json:"price,string"` // the limit price of limit order
	CreatedAt        int64      `json:"created-at"`   // the timestamp in milliseconds when the order was created
	AccountId        int64      `json:"account-id"`
	ClientOrderId    string     `json:"client-order-id"`           // client order id
	FilledAmount     float64    `json:"filled-amount,string"`      // the amount which has been filled
	FilledCashAmount float64    `json:"filled-cash-amount,string"` // the filled total in quote currency
	FilledFees       float64    `json:"filled-fees,string"`        // transaction fee
	Id               int64      `json:"id"`                        // order id
	State            OrderState `json:"state"`                     // order status (see above)
	OrderType        OrderType  `json:"type"`                      // order type (see above)
}

func (self *Order) Buy() bool {
	return self.OrderType == OrderTypeBuyMarket ||
		self.OrderType == OrderTypeBuyLimit ||
		self.OrderType == OrderTypeBuyIOC ||
		self.OrderType == OrderTypeBuyLimitMaker ||
		self.OrderType == OrderTypeBuyStopLimit ||
		self.OrderType == OrderTypeBuyLimitFOK ||
		self.OrderType == OrderTypeBuyStopLimitFOK
}

func (self *Order) Sell() bool {
	return !self.Buy()
}

func (self *Order) GetCreatedAt() time.Time {
	if self.CreatedAt == 0 {
		return time.Now().UTC()
	} else {
		return time.Unix((self.CreatedAt / 1000), 0)
	}
}

func (client *Client) OpenOrders(symbol string) ([]Order, error) {
	type Response struct {
		Data []Order `json:"data"`
	}

	var (
		err  error
		body []byte
		resp Response
	)

	params := url.Values{}
	params.Add("symbol", symbol)

	if body, err = client.get("/v1/order/openOrders", params, true); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}
