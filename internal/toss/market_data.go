package toss

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// OrderbookEntry is one price/volume level in an OrderbookResponse.
type OrderbookEntry struct {
	Price  string `json:"price"`
	Volume string `json:"volume"`
}

// OrderbookResponse is the result of GetOrderbook.
type OrderbookResponse struct {
	Timestamp *time.Time       `json:"timestamp"`
	Currency  string           `json:"currency"`
	Asks      []OrderbookEntry `json:"asks"`
	Bids      []OrderbookEntry `json:"bids"`
}

// GetOrderbook fetches the current bid/ask levels for symbol.
func (c *Client) GetOrderbook(ctx context.Context, symbol string) (OrderbookResponse, error) {
	query := map[string]string{"symbol": symbol}
	return doGet[OrderbookResponse](ctx, c, "toss orderbook", "/api/v1/orderbook", query, nil)
}

// PriceResponse is one entry of the result of GetPrices.
type PriceResponse struct {
	Symbol    string     `json:"symbol"`
	Timestamp *time.Time `json:"timestamp"`
	LastPrice string     `json:"lastPrice"`
	Currency  string     `json:"currency"`
}

// GetPrices fetches the current price for up to ~200 symbols in one call.
func (c *Client) GetPrices(ctx context.Context, symbols []string) ([]PriceResponse, error) {
	query := map[string]string{"symbols": strings.Join(symbols, ",")}
	return doGet[[]PriceResponse](ctx, c, "toss prices", "/api/v1/prices", query, nil)
}

// Trade is one recent execution in the result of GetTrades.
type Trade struct {
	Price     string    `json:"price"`
	Volume    string    `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
	Currency  string    `json:"currency"`
}

// GetTrades fetches the most recent trades for symbol. count<=0 omits the
// query param so the server applies its default (50, also the max).
func (c *Client) GetTrades(ctx context.Context, symbol string, count int) ([]Trade, error) {
	query := map[string]string{"symbol": symbol}
	if count > 0 {
		query["count"] = strconv.Itoa(count)
	}
	return doGet[[]Trade](ctx, c, "toss trades", "/api/v1/trades", query, nil)
}

// PriceLimitResponse is the result of GetPriceLimit.
type PriceLimitResponse struct {
	Timestamp       time.Time `json:"timestamp"`
	UpperLimitPrice *string   `json:"upperLimitPrice"`
	LowerLimitPrice *string   `json:"lowerLimitPrice"`
	Currency        string    `json:"currency"`
}

// GetPriceLimit fetches the current day's upper/lower price limits for
// symbol. Both limits are nil for markets without price limits (e.g. US
// stocks).
func (c *Client) GetPriceLimit(ctx context.Context, symbol string) (PriceLimitResponse, error) {
	query := map[string]string{"symbol": symbol}
	return doGet[PriceLimitResponse](ctx, c, "toss price-limits", "/api/v1/price-limits", query, nil)
}

// Candle is one OHLCV bar in a CandlePageResponse.
type Candle struct {
	Timestamp  time.Time `json:"timestamp"`
	OpenPrice  string    `json:"openPrice"`
	HighPrice  string    `json:"highPrice"`
	LowPrice   string    `json:"lowPrice"`
	ClosePrice string    `json:"closePrice"`
	Volume     string    `json:"volume"`
	Currency   string    `json:"currency"`
}

// CandlePageResponse is the result of GetCandles. NextBefore is nil on the
// last page; pass it as the before argument of the next call to paginate.
type CandlePageResponse struct {
	Candles    []Candle   `json:"candles"`
	NextBefore *time.Time `json:"nextBefore"`
}

// GetCandles fetches OHLCV candle data for symbol at the given interval
// ("1m" or "1d"). count<=0 omits the query param (server default 100, max
// 200). A zero before omits the query param (server returns the most recent
// page). A nil adjusted omits the query param (server default true);
// otherwise "true"/"false" is sent explicitly.
func (c *Client) GetCandles(
	ctx context.Context, symbol, interval string, count int, before time.Time, adjusted *bool,
) (CandlePageResponse, error) {
	query := map[string]string{"symbol": symbol, "interval": interval}
	if count > 0 {
		query["count"] = strconv.Itoa(count)
	}
	if !before.IsZero() {
		query["before"] = before.Format(time.RFC3339)
	}
	if adjusted != nil {
		query["adjusted"] = strconv.FormatBool(*adjusted)
	}
	return doGet[CandlePageResponse](ctx, c, "toss candles", "/api/v1/candles", query, nil)
}
