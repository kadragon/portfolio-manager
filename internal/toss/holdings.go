package toss

import "context"

// Price holds a value in KRW and, when applicable, its USD equivalent.
type Price struct {
	KRW string  `json:"krw"`
	USD *string `json:"usd"`
}

// MarketValue is a holding's purchase/current valuation.
type MarketValue struct {
	PurchaseAmount  string `json:"purchaseAmount"`
	Amount          string `json:"amount"`
	AmountAfterCost string `json:"amountAfterCost"`
}

// ProfitLoss is a holding's profit/loss, before and after trading costs.
type ProfitLoss struct {
	Amount          string `json:"amount"`
	AmountAfterCost string `json:"amountAfterCost"`
	Rate            string `json:"rate"`
	RateAfterCost   string `json:"rateAfterCost"`
}

// DailyProfitLoss is a holding's profit/loss for the current trading day.
type DailyProfitLoss struct {
	Amount string `json:"amount"`
	Rate   string `json:"rate"`
}

// Cost is the trading costs attributed to a holding.
type Cost struct {
	Commission string  `json:"commission"`
	Tax        *string `json:"tax"`
}

// HoldingsItem is one position within a HoldingsOverview.
type HoldingsItem struct {
	Symbol               string          `json:"symbol"`
	Name                 string          `json:"name"`
	MarketCountry        string          `json:"marketCountry"`
	Currency             string          `json:"currency"`
	Quantity             string          `json:"quantity"`
	LastPrice            string          `json:"lastPrice"`
	AveragePurchasePrice string          `json:"averagePurchasePrice"`
	MarketValue          MarketValue     `json:"marketValue"`
	ProfitLoss           ProfitLoss      `json:"profitLoss"`
	DailyProfitLoss      DailyProfitLoss `json:"dailyProfitLoss"`
	Cost                 Cost            `json:"cost"`
}

// OverviewMarketValue is the account-level total market value.
type OverviewMarketValue struct {
	Amount          Price `json:"amount"`
	AmountAfterCost Price `json:"amountAfterCost"`
}

// OverviewProfitLoss is the account-level total profit/loss.
type OverviewProfitLoss struct {
	Amount          Price  `json:"amount"`
	AmountAfterCost Price  `json:"amountAfterCost"`
	Rate            string `json:"rate"`
	RateAfterCost   string `json:"rateAfterCost"`
}

// OverviewDailyProfitLoss is the account-level daily profit/loss.
type OverviewDailyProfitLoss struct {
	Amount Price  `json:"amount"`
	Rate   string `json:"rate"`
}

// HoldingsOverview is the result of GetHoldings: account-level totals plus
// per-symbol items.
type HoldingsOverview struct {
	TotalPurchaseAmount Price                   `json:"totalPurchaseAmount"`
	MarketValue         OverviewMarketValue     `json:"marketValue"`
	ProfitLoss          OverviewProfitLoss      `json:"profitLoss"`
	DailyProfitLoss     OverviewDailyProfitLoss `json:"dailyProfitLoss"`
	Items               []HoldingsItem          `json:"items"`
}

// GetHoldings returns the raw holdings overview for accountSeq, optionally
// filtered to a single symbol. This is a faithful passthrough of the API;
// callers needing deduped/aggregated positions (e.g. account-sync) should
// post-process Items themselves.
func (c *Client) GetHoldings(ctx context.Context, accountSeq, symbol string) (HoldingsOverview, error) {
	query := map[string]string{}
	if symbol != "" {
		query["symbol"] = symbol
	}
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	return doGet[HoldingsOverview](ctx, c, "toss holdings", "/api/v1/holdings", query, headers)
}
