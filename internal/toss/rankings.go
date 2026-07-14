package toss

import (
	"context"
	"strconv"
	"time"
)

// RankingParams are the query parameters for GetRankings. Type,
// MarketCountry, and Duration are required by the API.
type RankingParams struct {
	// Type selects the ranking metric: MARKET_TRADING_AMOUNT,
	// MARKET_TRADING_VOLUME, TOP_GAINERS, TOP_LOSERS,
	// TOSS_SECURITIES_TRADING_AMOUNT, or TOSS_SECURITIES_TRADING_VOLUME.
	Type string
	// MarketCountry is the market to rank: KR or US.
	MarketCountry string
	// Duration is the ranking window: realtime, 1d, 1w, 1mo, 3mo, 6mo, or 1y.
	// TOP_GAINERS/TOP_LOSERS do not support realtime.
	Duration string
	// ExcludeInvestmentCaution excludes investment-caution stocks. The API
	// defaults to false, so it is only sent when true.
	ExcludeInvestmentCaution bool
	// Count caps the number of results (max 100, API default 100). Values
	// <= 0 omit the param and defer to the API default.
	Count int
}

// RankingPrice is the price snapshot for a ranked symbol.
type RankingPrice struct {
	LastPrice  string  `json:"lastPrice"`
	BasePrice  string  `json:"basePrice"`
	ChangeRate *string `json:"changeRate"`
}

// RankingItem is one ranked symbol.
type RankingItem struct {
	Rank          int          `json:"rank"`
	Symbol        string       `json:"symbol"`
	Currency      string       `json:"currency"`
	Price         RankingPrice `json:"price"`
	TradingVolume string       `json:"tradingVolume"`
	TradingAmount string       `json:"tradingAmount"`
}

// RankingResponse is the result of GetRankings.
type RankingResponse struct {
	RankedAt *time.Time    `json:"rankedAt"`
	Rankings []RankingItem `json:"rankings"`
}

// GetRankings returns the ranking list for params.Type/MarketCountry/Duration.
func (c *Client) GetRankings(ctx context.Context, params RankingParams) (RankingResponse, error) {
	query := map[string]string{
		"type":          params.Type,
		"marketCountry": params.MarketCountry,
		"duration":      params.Duration,
	}
	if params.ExcludeInvestmentCaution {
		query["excludeInvestmentCaution"] = "true"
	}
	if params.Count > 0 {
		query["count"] = strconv.Itoa(params.Count)
	}
	return doGet[RankingResponse](ctx, c, "toss rankings", "/api/v1/rankings", query, nil)
}
