package binance

import (
	"context"

	exchange "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/common"
)

type BookEntry = common.PriceLevel

func (self *Client) Depth(symbol string, limit int) (*exchange.DepthResponse, error) {
	defer AfterRequest()
	if limit < 500 {
		BeforeRequest(self, 1)
	} else if limit < 1000 {
		BeforeRequest(self, 5)
	} else if limit < 5000 {
		BeforeRequest(self, 10)
	} else {
		BeforeRequest(self, 50)
	}
	dept, err := self.inner.NewDepthService().Symbol(symbol).Limit(limit).Do(context.Background())
	self.handleError(err)
	return dept, err
}
