package bittrex

type MarketSummary struct {
	Symbol        string  `json:"symbol"`
	High          float64 `json:"high,string"`
	Low           float64 `json:"low,string"`
	Volume        float64 `json:"volume,string"`
	QuoteVolume   float64 `json:"quoteVolume,string"`
	PercentChange float64 `json:"percentChange,string"`
	UpdatedAt     string  `json:"updatedAt"`
}
