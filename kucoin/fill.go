package kucoin

import (
	"net/http"
	"strconv"
	"time"
)

// A FillModel represents the structure of fill.
type FillModel struct {
	Symbol         string `json:"symbol"`
	TradeId        string `json:"tradeId"`
	OrderId        string `json:"orderId"`
	CounterOrderId string `json:"counterOrderId"`
	Side           string `json:"side"`
	Liquidity      string `json:"liquidity"`
	ForceTaker     bool   `json:"forceTaker"`
	Price          string `json:"price"`
	Size           string `json:"size"`
	Funds          string `json:"funds"`
	Fee            string `json:"fee"`
	FeeRate        string `json:"feeRate"`
	FeeCurrency    string `json:"feeCurrency"`
	Stop           string `json:"stop"`
	Type           string `json:"type"`
	CreatedAt      int64  `json:"createdAt"`
}

// ParsePrice returns the price as float64
func (fill *FillModel) ParsePrice() float64 {
	out, err := strconv.ParseFloat(fill.Price, 64)
	if err == nil {
		return out
	}
	return 0
}

// ParseSize returns the quantity as float64
func (fill *FillModel) ParseSize() float64 {
	out, err := strconv.ParseFloat(fill.Size, 64)
	if err == nil {
		return out
	}
	return 0
}

// ParseCreatedAt returns the creation time as time.Time
func (fill *FillModel) ParseCreatedAt() time.Time {
	return time.Unix(fill.CreatedAt/1000, 0)
}

// A FillsModel is the set of *FillModel.
type FillsModel []*FillModel

func (fills FillsModel) IndexOfOrderId(orderId string) int {
	for idx, fill := range fills {
		if fill.OrderId == orderId {
			return idx
		}
	}
	return -1
}

// Fills returns a list of recent fills.
func (as *ApiService) Fills(params map[string]string, pagination *PaginationParam) (*ApiResponse, error) {
	pagination.ReadParam(params)
	req := NewRequest(http.MethodGet, "/api/v1/fills", params)
	return as.call(req, 3)
}

// RecentFills returns the recent fills of the latest transactions within 24 hours.
func (as *ApiService) RecentFills() (*ApiResponse, error) {
	req := NewRequest(http.MethodGet, "/api/v1/limit/fills", nil)
	return as.call(req, 3)
}
