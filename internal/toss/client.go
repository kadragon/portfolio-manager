// Package toss implements the Toss Securities Open API client pieces used by
// account holding sync.
package toss

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
)

const defaultBaseURL = "https://openapi.tossinvest.com"

// Client calls the Toss Securities Open API.
type Client struct {
	HTTP         *http.Client
	BaseURL      string
	ClientID     string
	ClientSecret string
	Now          func() time.Time

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// NewClient builds a Toss Open API client.
func NewClient(httpClient *http.Client, baseURL, clientID, clientSecret string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		HTTP:         httpClient,
		BaseURL:      strings.TrimRight(baseURL, "/"),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Now:          time.Now,
	}
}

// FetchAccountSnapshot fetches Toss holdings and cash buying power for
// accountSeq. KRW cash is added directly; USD cash is converted with Toss's
// current USD/KRW rate.
func (c *Client) FetchAccountSnapshot(accountSeq, _ string) (models.KisAccountSnapshot, error) {
	accountSeq = strings.TrimSpace(accountSeq)
	if accountSeq == "" {
		return models.KisAccountSnapshot{}, fmt.Errorf("toss: accountSeq is required")
	}
	ctx := context.Background()

	cashBalance, cashBalanceKRW, cashBalanceUSD, usdKRWRate, err := c.fetchCashBalances(ctx, accountSeq)
	if err != nil {
		return models.KisAccountSnapshot{}, err
	}

	overview, err := c.GetHoldings(ctx, accountSeq, "")
	if err != nil {
		return models.KisAccountSnapshot{}, err
	}

	return models.KisAccountSnapshot{
		CashBalance:    cashBalance,
		CashBalanceKRW: &cashBalanceKRW,
		CashBalanceUSD: &cashBalanceUSD,
		USDKRWRate:     usdKRWRate,
		Holdings:       aggregateHoldings(overview.Items),
	}, nil
}

// aggregateHoldings merges Toss holdings items by symbol (summing quantity),
// drops non-positive-quantity items, and sorts the result by symbol.
func aggregateHoldings(items []HoldingsItem) []models.KisHoldingPosition {
	bySymbol := map[string]models.KisHoldingPosition{}
	for _, item := range items {
		symbol := strings.ToUpper(strings.TrimSpace(item.Symbol))
		if symbol == "" {
			continue
		}
		qty := parseDecimal(item.Quantity)
		if !qty.IsPositive() {
			continue
		}
		pos := bySymbol[symbol]
		pos.Ticker = symbol
		pos.Quantity = numeric.Wrap(pos.Quantity.Decimal.Add(qty))
		if pos.Name == "" {
			pos.Name = strings.TrimSpace(item.Name)
		}
		bySymbol[symbol] = pos
	}

	symbols := make([]string, 0, len(bySymbol))
	for symbol := range bySymbol {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)

	holdings := make([]models.KisHoldingPosition, 0, len(symbols))
	for _, symbol := range symbols {
		holdings = append(holdings, bySymbol[symbol])
	}
	return holdings
}

// fetchCashBalances returns (total KRW-equivalent cash, KRW cash, USD cash,
// USD/KRW rate used for the conversion [nil if USD cash is zero]).
func (c *Client) fetchCashBalances(
	ctx context.Context, accountSeq string,
) (numeric.Decimal, numeric.Decimal, numeric.Decimal, *numeric.Decimal, error) {
	krw, err := c.buyingPowerDecimal(ctx, accountSeq, "KRW")
	if err != nil {
		return numeric.Decimal{}, numeric.Decimal{}, numeric.Decimal{}, nil, err
	}
	usd, err := c.buyingPowerDecimal(ctx, accountSeq, "USD")
	if err != nil {
		return numeric.Decimal{}, numeric.Decimal{}, numeric.Decimal{}, nil, err
	}
	if usd.IsZero() {
		return krw, krw, usd, nil, nil
	}
	rate, err := c.usdKRWRateDecimal(ctx)
	if err != nil {
		return numeric.Decimal{}, numeric.Decimal{}, numeric.Decimal{}, nil, err
	}
	total := numeric.Wrap(krw.Decimal.Add(usd.Decimal.Mul(rate.Decimal)))
	return total, krw, usd, &rate, nil
}

func (c *Client) buyingPowerDecimal(ctx context.Context, accountSeq, currency string) (numeric.Decimal, error) {
	resp, err := c.GetBuyingPower(ctx, accountSeq, currency)
	if err != nil {
		return numeric.Decimal{}, err
	}
	if !strings.EqualFold(resp.Currency, currency) {
		return numeric.Decimal{}, fmt.Errorf("toss buying-power: unexpected currency %q", resp.Currency)
	}
	power := numeric.Wrap(parseDecimal(resp.CashBuyingPower))
	if power.IsNegative() {
		return numeric.Decimal{}, fmt.Errorf("toss buying-power: negative %s cashBuyingPower %q", currency, resp.CashBuyingPower)
	}
	return power, nil
}

func (c *Client) usdKRWRateDecimal(ctx context.Context) (numeric.Decimal, error) {
	resp, err := c.GetExchangeRate(ctx, "USD", "KRW", time.Time{})
	if err != nil {
		return numeric.Decimal{}, err
	}
	if !strings.EqualFold(resp.BaseCurrency, "USD") || !strings.EqualFold(resp.QuoteCurrency, "KRW") {
		return numeric.Decimal{}, fmt.Errorf("toss exchange-rate: unexpected pair %q/%q", resp.BaseCurrency, resp.QuoteCurrency)
	}
	rate := parseDecimal(resp.Rate)
	if !rate.IsPositive() {
		return numeric.Decimal{}, fmt.Errorf("toss exchange-rate: invalid USD/KRW rate %q", resp.Rate)
	}
	return numeric.Wrap(rate), nil
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.Now()
	if c.token != "" && now.Before(c.expiresAt.Add(-time.Minute)) {
		return c.token, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.ClientID)
	form.Set("client_secret", c.ClientSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/oauth2/token", strings.NewReader(form.Encode())) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return "", fmt.Errorf("toss auth: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTP.Do(req) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return "", fmt.Errorf("toss auth: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("toss auth: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", parseOAuthError(resp.StatusCode, body)
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("toss auth: json unmarshal: %w", err)
	}
	if !strings.EqualFold(tr.TokenType, "Bearer") || tr.AccessToken == "" {
		return "", fmt.Errorf("toss auth: invalid token response")
	}
	c.token = tr.AccessToken
	c.expiresAt = now.Add(time.Duration(tr.ExpiresIn) * time.Second)
	return c.token, nil
}

func parseDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return decimal.Zero
	}
	return d
}
