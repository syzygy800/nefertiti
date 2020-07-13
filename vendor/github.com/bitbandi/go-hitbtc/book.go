package hitbtc

type BookEntry struct {
	Price float64 `json:"price,string"`
	Size  float64 `json:"size,string"`
}

type Book struct {
	Ask []BookEntry `json:"ask"`
	Bid []BookEntry `json:"bid"`
}
