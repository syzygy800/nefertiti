package binance

import (
	"github.com/adshao/go-binance/v2/common"
)

type BinanceError = common.APIError

func IsBinanceError(err error) (*BinanceError, bool) {
	if err == nil {
		return nil, false
	}
	apiError, ok := err.(*common.APIError)
	if ok {
		return apiError, true
	}
	return nil, false
}
