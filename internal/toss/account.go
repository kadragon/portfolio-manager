package toss

import "context"

// Account is a Toss brokerage account. AccountSeq is the identifier used in
// the X-Tossinvest-Account header for all other user-context APIs (holdings,
// orders, buying-power, ...).
type Account struct {
	AccountNo   string `json:"accountNo"`
	AccountSeq  int64  `json:"accountSeq"`
	AccountType string `json:"accountType"` // e.g. BROKERAGE/OVERSEAS_DERIVATIVES/PENSION_SAVINGS/RESHORING_INVESTMENT
}

// GetAccounts returns the caller's Toss brokerage accounts. Only BROKERAGE
// accounts are currently returned by the API; sub-accounts are not exposed.
func (c *Client) GetAccounts(ctx context.Context) ([]Account, error) {
	return doGet[[]Account](ctx, c, "toss accounts", "/api/v1/accounts", nil, nil)
}
