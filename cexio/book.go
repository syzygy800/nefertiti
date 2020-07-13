package cexio

type (
	BookEntry []float64
)

func (be *BookEntry) Price() float64 {
	return (*be)[0]
}

func (be *BookEntry) Size() float64 {
	return (*be)[1]
}

type OrderBook struct {
	Timestamp int64       `json:"timestamp"`
	Bids      []BookEntry `json:"bids"`
	Asks      []BookEntry `json:"asks"`
	Pair      string      `json:"pair"` // warning: NOT equal to market name
	Id        int64       `json:"id"`
	SellTotal float64     `json:"sell_total,string"`
	BuyTotal  float64     `json:"buy_total,string"`
}
