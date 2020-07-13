package cexio

type (
	Pair struct {
		MaxLotSize float64 `json:"maxLotSize"`
		MinLotSize float64 `json:"minLotSize"`
		MaxPrice   float64 `json:"maxPrice,string"`
		MinPrice   float64 `json:"minPrice,string"`
		Symbol1    string  `json:"symbol1"` // 1st listed currency of this market pair
		Symbol2    string  `json:"symbol2"` // 2nd listed currency of this market pair
	}
)
