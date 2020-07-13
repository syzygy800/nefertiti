package bittrex

type MarketSummary1 struct {
	MarketName     string  `json:"MarketName"`
	High           float64 `json:"High"`
	Low            float64 `json:"Low"`
	Ask            float64 `json:"Ask"`
	Bid            float64 `json:"Bid"`
	OpenBuyOrders  int     `json:"OpenBuyOrders"`
	OpenSellOrders int     `json:"OpenSellOrders"`
	Volume         float64 `json:"Volume"`
	Last           float64 `json:"Last"`
	BaseVolume     float64 `json:"BaseVolume"`
	PrevDay        float64 `json:"PrevDay"`
	TimeStamp      string  `json:"TimeStamp"`
}

type MarketSummary3 struct {
	Symbol        string  `json:"symbol"`
	High          float64 `json:"high,string"`
	Low           float64 `json:"low,string"`
	Volume        float64 `json:"volume,string"`
	QuoteVolume   float64 `json:"quoteVolume,string"`
	PercentChange float64 `json:"percentChange,string"`
	UpdatedAt     string  `json:"updatedAt"`
}
