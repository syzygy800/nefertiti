package bittrex

type Ticker1 struct {
	Bid  float64 `json:"Bid"`
	Ask  float64 `json:"Ask"`
	Last float64 `json:"Last"`
}

type Ticker3 struct {
	Symbol        string  `json:"symbol"`
	LastTradeRate float64 `json:"lastTradeRate,string"`
	BidRate       float64 `json:"bidRate,string"`
	AskRate       float64 `json:"askRate,string"`
}
