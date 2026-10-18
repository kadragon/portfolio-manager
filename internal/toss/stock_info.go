package toss

import (
	"context"
	"net/url"
	"strconv"
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

// ListStocksParams are the query parameters for ListStocks. Market is
// required by the API; the rest are optional filters.
type ListStocksParams struct {
	// Market is KOSPI, KOSDAQ, NYSE, NASDAQ, AMEX, KR_ETC, or US_ETC.
	Market string
	// Status is SCHEDULED, ACTIVE, or DELISTED. Empty omits the param; the
	// API then defaults to ACTIVE.
	Status string
	// SecurityType filters by security type (STOCK, ETF, ETN, REIT, ...).
	// Empty omits the param and returns every type.
	SecurityType string
	// CommonShare filters common (true) or preferred (false) shares; nil
	// omits the param and returns both.
	CommonShare *bool
}

// ListedStock is one entry of the market-wide universe returned by
// ListStocks.
type ListedStock struct {
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	SecurityType  string `json:"securityType"`
	IsCommonShare bool   `json:"isCommonShare"`
	IsinCode      string `json:"isinCode"`
}

// ListStocks returns every Toss-tradable symbol on params.Market matching
// the optional filters, sorted by symbol, unpaginated (up to a few thousand
// entries). The data is refreshed daily.
func (c *Client) ListStocks(ctx context.Context, params ListStocksParams) ([]ListedStock, error) {
	query := map[string]string{
		"market":       params.Market,
		"status":       params.Status,
		"securityType": params.SecurityType,
	}
	if params.CommonShare != nil {
		query["commonShare"] = strconv.FormatBool(*params.CommonShare)
	}
	return doGet[[]ListedStock](ctx, c, "toss stocks-all", "/api/v1/stocks/all", query, nil)
}
