package ethplorer

type (
	Token struct {
		Address string `json:"address"`
		Name    string `json:"name"`
		Symbol  string `json:"symbol"`
		Price   Price  `json:"price"`
	}
)
