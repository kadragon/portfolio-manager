package toss

import (
	"context"
	"net/url"
	"strings"
)

// KrMarketDetail is stock reference data specific to Korean-listed
// securities (KOSPI/KOSDAQ/KR_ETC). Present only when StockInfo.Market is a
// Korean market; nil for overseas listings.
type KrMarketDetail struct {
	LiquidationTrading  bool  `json:"liquidationTrading"`
	NxtSupported        bool  `json:"nxtSupported"`
	KrxTradingSuspended bool  `json:"krxTradingSuspended"`
	NxtTradingSuspended *bool `json:"nxtTradingSuspended"`
}

// StockInfo is one symbol's reference data, as returned by GetStocks.
type StockInfo struct {
	Symbol             string          `json:"symbol"`
	Name               string          `json:"name"`
	EnglishName        string          `json:"englishName"`
	IsinCode           string          `json:"isinCode"`
	Market             string          `json:"market"` // KOSPI/KOSDAQ/NYSE/NASDAQ/AMEX/KR_ETC/US_ETC
	SecurityType       string          `json:"securityType"`
	IsCommonShare      bool            `json:"isCommonShare"`
	Status             string          `json:"status"` // SCHEDULED/ACTIVE/DELISTED
	Currency           string          `json:"currency"`
	ListDate           *string         `json:"listDate"`
	DelistDate         *string         `json:"delistDate"`
	SharesOutstanding  string          `json:"sharesOutstanding"`
	LeverageFactor     *string         `json:"leverageFactor"`
	KoreanMarketDetail *KrMarketDetail `json:"koreanMarketDetail"`
}

// StockWarning is one active or historical trading caution/VI event for a
// symbol, as returned by GetStockWarnings.
type StockWarning struct {
	WarningType string  `json:"warningType"`
	Exchange    *string `json:"exchange"`
	StartDate   *string `json:"startDate"`
	EndDate     *string `json:"endDate"`
}

// GetStocks returns reference data for up to ~200 symbols in one call.
func (c *Client) GetStocks(ctx context.Context, symbols []string) ([]StockInfo, error) {
	query := map[string]string{"symbols": strings.Join(symbols, ",")}
	return doGet[[]StockInfo](ctx, c, "toss stocks", "/api/v1/stocks", query, nil)
}

// GetStockWarnings returns active/historical trading-caution and VI events
// for symbol.
func (c *Client) GetStockWarnings(ctx context.Context, symbol string) ([]StockWarning, error) {
	path := "/api/v1/stocks/" + url.PathEscape(symbol) + "/warnings"
	return doGet[[]StockWarning](ctx, c, "toss stock-warnings", path, nil, nil)
}
