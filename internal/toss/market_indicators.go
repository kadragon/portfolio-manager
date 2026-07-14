package toss

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// MarketIndicatorPriceResponse is one entry of GetMarketIndicatorPrices.
type MarketIndicatorPriceResponse struct {
	Symbol    string     `json:"symbol"`
	Timestamp *time.Time `json:"timestamp"`
	LastPrice string     `json:"lastPrice"`
}

// GetMarketIndicatorPrices returns the latest price for each of symbols
// (index/bond symbols such as KOSPI, KOSDAQ — not stock tickers).
func (c *Client) GetMarketIndicatorPrices(ctx context.Context, symbols []string) ([]MarketIndicatorPriceResponse, error) {
	query := map[string]string{"symbols": strings.Join(symbols, ",")}
	return doGet[[]MarketIndicatorPriceResponse](ctx, c, "toss market-indicators prices", "/api/v1/market-indicators/prices", query, nil)
}

// MarketIndicatorCandle is one OHLCV candle for a market indicator.
type MarketIndicatorCandle struct {
	Timestamp  time.Time `json:"timestamp"`
	OpenPrice  string    `json:"openPrice"`
	HighPrice  string    `json:"highPrice"`
	LowPrice   string    `json:"lowPrice"`
	ClosePrice string    `json:"closePrice"`
	Volume     string    `json:"volume"`
}

// MarketIndicatorCandlePageResponse is the result of GetMarketIndicatorCandles.
type MarketIndicatorCandlePageResponse struct {
	Candles    []MarketIndicatorCandle `json:"candles"`
	NextBefore *time.Time              `json:"nextBefore"`
}

// GetMarketIndicatorCandles returns candles for symbol at interval ("1m" or
// "1d"). count caps the number of candles returned (max 200); values <= 0
// omit the param and defer to the API default. before is an inclusive upper
// bound for pagination (pass the previous response's NextBefore); a zero
// value omits the param and returns the most recent candles.
func (c *Client) GetMarketIndicatorCandles(ctx context.Context, symbol, interval string, count int, before time.Time) (MarketIndicatorCandlePageResponse, error) {
	query := map[string]string{"interval": interval}
	if count > 0 {
		query["count"] = strconv.Itoa(count)
	}
	if !before.IsZero() {
		query["before"] = before.Format(time.RFC3339)
	}
	path := "/api/v1/market-indicators/" + url.PathEscape(symbol) + "/candles"
	return doGet[MarketIndicatorCandlePageResponse](ctx, c, "toss market-indicators candles", path, query, nil)
}

// InvestorTradingAmount is a buy/sell trading amount pair, in KRW.
type InvestorTradingAmount struct {
	BuyAmount  string `json:"buyAmount"`
	SellAmount string `json:"sellAmount"`
}

// InstitutionTradingBreakdown splits institutional trading into its 7
// sub-categories.
type InstitutionTradingBreakdown struct {
	FinancialInvestment       InvestorTradingAmount `json:"financialInvestment"`
	Insurance                 InvestorTradingAmount `json:"insurance"`
	Trust                     InvestorTradingAmount `json:"trust"`
	PrivateEquityFund         InvestorTradingAmount `json:"privateEquityFund"`
	Bank                      InvestorTradingAmount `json:"bank"`
	OtherFinancialInstitution InvestorTradingAmount `json:"otherFinancialInstitution"`
	PensionFund               InvestorTradingAmount `json:"pensionFund"`
}

// InstitutionTradingAmount is the institutional trading total plus its
// per-category breakdown.
type InstitutionTradingAmount struct {
	BuyAmount  string                      `json:"buyAmount"`
	SellAmount string                      `json:"sellAmount"`
	Breakdown  InstitutionTradingBreakdown `json:"breakdown"`
}

// InvestorTradingRecord is one aggregation-period trading record.
type InvestorTradingRecord struct {
	Date             string                   `json:"date"`
	UpdatedAt        time.Time                `json:"updatedAt"`
	Individual       InvestorTradingAmount    `json:"individual"`
	Foreigner        InvestorTradingAmount    `json:"foreigner"`
	Institution      InstitutionTradingAmount `json:"institution"`
	OtherCorporation InvestorTradingAmount    `json:"otherCorporation"`
}

// InvestorTradingResponse is the result of GetMarketIndicatorInvestorTrading.
type InvestorTradingResponse struct {
	NextUntil *string                 `json:"nextUntil"`
	Records   []InvestorTradingRecord `json:"records"`
}

// GetMarketIndicatorInvestorTrading returns investor-trading records for
// symbol, which per the API spec must be "KOSPI" or "KOSDAQ". interval is
// the aggregation unit ("1d", "1w", "1mo", or "1y"). count caps the number
// of records returned (max 100); values <= 0 omit the param and defer to the
// API default. until is an inclusive upper bound date ("YYYY-MM-DD") for
// pagination (pass the previous response's NextUntil); an empty string
// omits the param and returns the most recent records.
func (c *Client) GetMarketIndicatorInvestorTrading(ctx context.Context, symbol, interval string, count int, until string) (InvestorTradingResponse, error) {
	if symbol != "KOSPI" && symbol != "KOSDAQ" {
		return InvestorTradingResponse{}, fmt.Errorf("toss investor-trading: symbol must be KOSPI or KOSDAQ, got %q", symbol)
	}
	query := map[string]string{"interval": interval}
	if count > 0 {
		query["count"] = strconv.Itoa(count)
	}
	if until != "" {
		query["until"] = until
	}
	path := "/api/v1/market-indicators/" + url.PathEscape(symbol) + "/investor-trading"
	return doGet[InvestorTradingResponse](ctx, c, "toss market-indicators investor-trading", path, query, nil)
}
