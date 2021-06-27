package bittrex

type OrderBook struct {
	Bid []BookEntry `json:"bid"`
	Ask []BookEntry `json:"ask"`
}

type BookEntry struct {
	Quantity float64 `json:"quantity,string"`
	Rate     float64 `json:"rate,string"`
}
