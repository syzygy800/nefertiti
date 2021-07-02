package kucoin

import (
	"net/http"
)

// An AccountModel represents an account.
type AccountModel struct {
	Id        string `json:"id"`
	Currency  string `json:"currency"`
	Type      string `json:"type"`
	Balance   string `json:"balance"`
	Available string `json:"available"`
	Holds     string `json:"holds"`
}

// An AccountsModel is the set of *AccountModel.
type AccountsModel []*AccountModel

// Accounts returns a list of accounts.
// See the Deposits section for documentation on how to deposit funds to begin trading.
func (as *ApiService) Accounts(currency, typo string) (*ApiResponse, error) {
	p := map[string]string{}
	if currency != "" {
		p["currency"] = currency
	}
	if typo != "" {
		p["type"] = typo
	}
	req := NewRequest(http.MethodGet, "/api/v1/accounts", p)
	return as.Call(req)
}

// Account returns an account when you know the accountId.
func (as *ApiService) Account(accountId string) (*ApiResponse, error) {
	req := NewRequest(http.MethodGet, "/api/v1/accounts/"+accountId, nil)
	return as.Call(req)
}
