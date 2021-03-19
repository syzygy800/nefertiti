package kucoin

import (
	"net/http"
	"strconv"
)

// A SymbolModel represents an available currency pairs for trading.
type SymbolModel struct {
	Symbol         string `json:"symbol"`
	Name           string `json:"name"`
	BaseCurrency   string `json:"baseCurrency"`
	QuoteCurrency  string `json:"quoteCurrency"`
	BaseMinSize    string `json:"baseMinSize"`
	QuoteMinSize   string `json:"quoteMinSize"`
	BaseMaxSize    string `json:"baseMaxSize"`
	QuoteMaxSize   string `json:"quoteMaxSize"`
	BaseIncrement  string `json:"baseIncrement"`
	QuoteIncrement string `json:"quoteIncrement"`
	PriceIncrement string `json:"priceIncrement"`
	EnableTrading  bool   `json:"enableTrading"`
}

// A SymbolsModel is the set of *SymbolModel.
type SymbolsModel []*SymbolModel

// Symbols returns a list of available currency pairs for trading.
func (as *ApiService) Symbols(market string) (*ApiResponse, error) {
	p := map[string]string{}
	if market != "" {
		p["market"] = market
	}
	req := NewRequest(http.MethodGet, "/api/v1/symbols", p)
	return as.Call(req)
}

// A TickerLevel1Model represents ticker include only the inside (i.e. best) bid and ask data, last price and last trade size.
type TickerLevel1Model struct {
	Sequence    string `json:"sequence"`
	Price       string `json:"price"`
	Size        string `json:"size"`
	BestBid     string `json:"bestBid"`
	BestBidSize string `json:"bestBidSize"`
	BestAsk     string `json:"bestAsk"`
	BestAskSize string `json:"bestAskSize"`
	Time        int64  `json:"time"`
}

// TickerLevel1 returns the ticker include only the inside (i.e. best) bid and ask data, last price and last trade size.
func (as *ApiService) TickerLevel1(symbol string) (*ApiResponse, error) {
	req := NewRequest(http.MethodGet, "/api/v1/market/orderbook/level1", map[string]string{"symbol": symbol})
	return as.Call(req)
}

// A TickerModel represents a market ticker for all trading pairs in the market (including 24h volume).
type TickerModel struct {
	Symbol      string `json:"symbol"`
	Buy         string `json:"buy"`
	Sell        string `json:"sell"`
	ChangeRate  string `json:"changeRate"`
	ChangePrice string `json:"changePrice"`
	High        string `json:"high"`
	Low         string `json:"low"`
	Vol         string `json:"vol"`
	Last        string `json:"last"`
}

// A TickersModel is the set of *MarketTickerModel.
type TickersModel []*TickerModel

// TickersResponseModel represents the response model of MarketTickers().
type TickersResponseModel struct {
	Time    int64        `json:"time"`
	Tickers TickersModel `json:"ticker"`
}

// Tickers returns all tickers as TickersResponseModel for all trading pairs in the market (including 24h volume).
func (as *ApiService) Tickers() (*ApiResponse, error) {
	req := NewRequest(http.MethodGet, "/api/v1/market/allTickers", nil)
	return as.Call(req)
}

// A Stats24hrModel represents 24 hr stats for the symbol.
// Volume is in base currency units.
// Open, high, low are in quote currency units.
type Stats24hrModel struct {
	Symbol       string `json:"symbol"`
	High         string `json:"high"`
	Vol          string `json:"vol"`
	VolValue     string `json:"volValue"`
	Last         string `json:"last"`
	Low          string `json:"low"`
	Buy          string `json:"buy"`
	Sell         string `json:"sell"`
	ChangeRate   string `json:"changeRate"`
	AveragePrice string `json:"averagePrice"`
	ChangePrice  string `json:"changePrice"`
}

// Stats24hr returns 24 hr stats for the symbol. volume is in base currency units. open, high, low are in quote currency units.
func (as *ApiService) Stats24hr(symbol string) (*ApiResponse, error) {
	req := NewRequest(http.MethodGet, "/api/v1/market/stats", map[string]string{"symbol": symbol})
	return as.Call(req)
}

// Markets returns the transaction currencies for the entire trading market.
func (as *ApiService) Markets() (*ApiResponse, error) {
	req := NewRequest(http.MethodGet, "/api/v1/markets", nil)
	return as.Call(req)
}

// BookEntry = bid or ask info with price and size
type BookEntry []string

// Price = bid or ask price
func (be *BookEntry) Price() float64 {
	out, _ := strconv.ParseFloat((*be)[0], 64)
	return out
}

// Size = bid or ask size
func (be *BookEntry) Size() float64 {
	out, _ := strconv.ParseFloat((*be)[1], 64)
	return out
}

// A FullOrderBookModel represents a list of open orders for a symbol, with full depth.
type FullOrderBookModel struct {
	Sequence string      `json:"sequence"`
	Bids     []BookEntry `json:"bids"`
	Asks     []BookEntry `json:"asks"`
}

// AggregatedFullOrderBook returns a list of open orders(aggregated) for a symbol.
func (as *ApiService) AggregatedFullOrderBook(symbol string) (*ApiResponse, error) {
	req := NewRequest(http.MethodGet, "/api/v2/market/orderbook/level2", map[string]string{"symbol": symbol})
	return as.Call(req)
}

// A TradeHistoryModel represents a the latest trades for a symbol.
type TradeHistoryModel struct {
	Sequence string `json:"sequence"`
	Price    string `json:"price"`
	Size     string `json:"size"`
	Side     string `json:"side"`
	Time     int64  `json:"time"`
}

// A TradeHistoriesModel is the set of *TradeHistoryModel.
type TradeHistoriesModel []*TradeHistoryModel

// TradeHistories returns a list the latest trades for a symbol.
func (as *ApiService) TradeHistories(symbol string) (*ApiResponse, error) {
	req := NewRequest(http.MethodGet, "/api/v1/market/histories", map[string]string{"symbol": symbol})
	return as.Call(req)
}

// KLineModel represents the k lines for a symbol.
// Rates are returned in grouped buckets based on requested type.
type KLineModel []string

// A KLinesModel is the set of *KLineModel.
type KLinesModel []*KLineModel

// KLines returns the k lines for a symbol.
// Data are returned in grouped buckets based on requested type.
// Parameter #2 typo is the type of candlestick patterns.
func (as *ApiService) KLines(symbol, typo string, startAt, endAt int64) (*ApiResponse, error) {
	req := NewRequest(http.MethodGet, "/api/v1/market/candles", map[string]string{
		"symbol":  symbol,
		"type":    typo,
		"startAt": IntToString(startAt),
		"endAt":   IntToString(endAt),
	})
	return as.Call(req)
}
