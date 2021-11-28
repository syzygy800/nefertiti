package huobi

import (
	"encoding/json"
	"fmt"
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
	OrderTypeBuyIOC           OrderType = "buy-ioc"  // IOC stands for "immediately or cancel", it means the order will be canceled if it couldn't be matched. If the order is partially traded, the remaining part will be canceled.
	OrderTypeSellIOC          OrderType = "sell-ioc" // IOC stands for "immediately or cancel", it means the order will be canceled if it couldn't be matched. If the order is partially traded, the remaining part will be canceled.
	OrderTypeBuyLimitMaker    OrderType = "buy-limit-maker"
	OrderTypeSellLimitMaker   OrderType = "sell-limit-maker"
	OrderTypeBuyStopLimit     OrderType = "buy-stop-limit"
	OrderTypeSellStopLimit    OrderType = "sell-stop-limit"
	OrderTypeBuyLimitFOK      OrderType = "buy-limit-fok"  // FOK stands for "fill or kill", it means the order will be cancelled if it couldn't be fully matched. Even if the order can be partially filled, the entire order will be cancelled.
	OrderTypeSellLimitFOK     OrderType = "sell-limit-fok" // FOK stands for "fill or kill", it means the order will be cancelled if it couldn't be fully matched. Even if the order can be partially filled, the entire order will be cancelled.
	OrderTypeBuyStopLimitFOK  OrderType = "buy-stop-limit-fok"
	OrderTypeSellStopLimitFOK OrderType = "sell-stop-limit-fok"
)

const (
	OrderStateCreated         OrderState = "created"          // The order is created, and not in the matching queue yet.
	OrderStateSubmitted       OrderState = "submitted"        // The order is submitted, and already in the matching queue, waiting for deal.
	OrderStatePartialFilled   OrderState = "partial-filled"   // The order is already in the matching queue and partially traded, and is waiting for further matching and trade.
	OrderStateFilled          OrderState = "filled"           // The order is already traded and not in the matching queue any more.
	OrderStatePartialCanceled OrderState = "partial-canceled" // The order is not in the matching queue any more. The status is transferred from 'partial-filled', the order is partially trade, but remaining is canceled.
	OrderStateCanceling       OrderState = "canceling"        // The order is under canceling, but haven't been removed from matching queue yet.
	OrderStateCanceled        OrderState = "canceled"         // The order is not in the matching queue any more, and completely canceled. There is no trade associated with this order.
)

type Order struct {
	Symbol           string     `json:"symbol"` // the trading symbol to trade, e.g. btcusdt, bccbtc
	Amount           float64    `json:"amount,string"`
	Price            float64    `json:"price,string"` // the limit price of limit order
	CreatedAt        int64      `json:"created-at"`   // the timestamp in milliseconds when the order was created
	AccountId        int64      `json:"account-id"`
	ClientOrderId    string     `json:"client-order-id"`           // the identity defined by the client
	FilledAmount     float64    `json:"filled-amount,string"`      // the amount which has been filled
	FilledCashAmount float64    `json:"filled-cash-amount,string"` // the filled total in quote currency
	FilledFees       float64    `json:"filled-fees,string"`        // transaction fee
	Id               int64      `json:"id"`                        // the unique identity for the order
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

func (client *Client) CancelOrder(orderId int64) error {
	_, err := client.post(fmt.Sprintf("/v1/order/orders/%d/submitcancel", orderId), nil)
	return err
}
