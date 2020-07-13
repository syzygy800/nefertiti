package bittrex

type OrderBook1 struct {
	Buy  []BookEntry1 `json:"buy"`
	Sell []BookEntry1 `json:"sell"`
}

type BookEntry1 struct {
	Quantity float64 `json:"Quantity"`
	Rate     float64 `json:"Rate"`
}

type OrderBook3 struct {
	Bid []BookEntry3 `json:"bid"`
	Ask []BookEntry3 `json:"ask"`
}

type BookEntry3 struct {
	Quantity float64 `json:"quantity,string"`
	Rate     float64 `json:"rate,string"`
}
