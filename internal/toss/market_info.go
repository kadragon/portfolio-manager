package toss

import (
	"context"
	"time"
)

// ExchangeRateResponse is the result of GetExchangeRate.
type ExchangeRateResponse struct {
	BaseCurrency   string    `json:"baseCurrency"`
	QuoteCurrency  string    `json:"quoteCurrency"`
	Rate           string    `json:"rate"`
	MidRate        string    `json:"midRate"`
	BasisPoint     string    `json:"basisPoint"`
	RateChangeType string    `json:"rateChangeType"`
	ValidFrom      time.Time `json:"validFrom"`
	ValidUntil     time.Time `json:"validUntil"`
}

// GetExchangeRate fetches the exchange rate between base and quote
// currencies. A zero at omits the dateTime query param, returning the
// current rate.
func (c *Client) GetExchangeRate(ctx context.Context, base, quote string, at time.Time) (ExchangeRateResponse, error) {
	query := map[string]string{
		"baseCurrency":  base,
		"quoteCurrency": quote,
	}
	if !at.IsZero() {
		query["dateTime"] = at.Format(time.RFC3339)
	}
	return doGet[ExchangeRateResponse](ctx, c, "toss exchange-rate", "/api/v1/exchange-rate", query, nil)
}
