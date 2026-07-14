package toss

import "context"

// BuyingPowerResponse is the result of GetBuyingPower.
type BuyingPowerResponse struct {
	Currency        string `json:"currency"`
	CashBuyingPower string `json:"cashBuyingPower"`
}

// GetBuyingPower returns the cash buying power for accountSeq in currency
// ("KRW" or "USD"). This is a faithful passthrough of the API; callers
// needing validated/parsed amounts (e.g. account-sync) should parse
// CashBuyingPower themselves.
func (c *Client) GetBuyingPower(ctx context.Context, accountSeq, currency string) (BuyingPowerResponse, error) {
	query := map[string]string{"currency": currency}
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	return doGet[BuyingPowerResponse](ctx, c, "toss buying-power", "/api/v1/buying-power", query, headers)
}
