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

// SellableQuantityResponse is the result of GetSellableQuantity.
type SellableQuantityResponse struct {
	SellableQuantity string `json:"sellableQuantity"`
}

// GetSellableQuantity returns the sellable quantity of symbol for accountSeq
// (integer shares for KR, fractional allowed for US).
func (c *Client) GetSellableQuantity(ctx context.Context, accountSeq, symbol string) (SellableQuantityResponse, error) {
	query := map[string]string{"symbol": symbol}
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	return doGet[SellableQuantityResponse](ctx, c, "toss sellable-quantity", "/api/v1/sellable-quantity", query, headers)
}

// Commission is a per-market commission rate entry returned by GetCommissions.
type Commission struct {
	MarketCountry  string  `json:"marketCountry"` // KR/US
	CommissionRate string  `json:"commissionRate"`
	StartDate      *string `json:"startDate"`
	EndDate        *string `json:"endDate"`
}

// GetCommissions returns the current per-market commission rates for accountSeq.
func (c *Client) GetCommissions(ctx context.Context, accountSeq string) ([]Commission, error) {
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	return doGet[[]Commission](ctx, c, "toss commissions", "/api/v1/commissions", nil, headers)
}
