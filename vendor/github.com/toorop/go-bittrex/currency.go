package bittrex

type Currency1 struct {
	Currency        string  `json:"Currency"`
	CurrencyLong    string  `json:"CurrencyLong"`
	MinConfirmation int     `json:"MinConfirmation"`
	TxFee           float64 `json:"TxFee"`
	IsActive        bool    `json:"IsActive"`
	CoinType        string  `json:"CoinType"`
}
