package ethplorer

type Criteria int

const (
	ByTradeVolume Criteria = iota
	ByCapitalization
	ByOperations
)

var CriteriaString = map[Criteria]string{
	ByTradeVolume:    "trade",
	ByCapitalization: "cap",
	ByOperations:     "count",
}

func (c *Criteria) String() string {
	return CriteriaString[*c]
}

func NewCriteria(data string) Criteria {
	for c := range CriteriaString {
		if c.String() == data {
			return c
		}
	}
	return ByTradeVolume
}
