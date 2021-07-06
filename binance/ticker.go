package binance

import (
	"context"
	"fmt"

	exchange "github.com/adshao/go-binance/v2"
)

// 24 hour rolling window price change statistics.
func (self *Client) Ticker(symbol string) (*exchange.PriceChangeStats, error) {
	var (
		err   error
		stats []*exchange.PriceChangeStats
	)
	defer AfterRequest()
	BeforeRequest(self, WEIGHT_TICKER_24H_WITH_SYMBOL)
	if stats, err = self.inner.NewListPriceChangeStatsService().Symbol(symbol).Do(context.Background()); err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return nil, fmt.Errorf("symbol %s does not exist", symbol)
	}
	return stats[0], nil
}
