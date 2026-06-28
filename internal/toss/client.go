// Package toss implements the Toss Securities Open API client pieces used by
// account holding sync.
package toss

import (
	"bytes"
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

type apiErrorResponse struct {
	Error struct {
		RequestID string `json:"requestId"`
		Code      string `json:"code"`
		Message   string `json:"message"`
	} `json:"error"`
}

type holdingsResponse struct {
	Result struct {
		Items []holdingItem `json:"items"`
	} `json:"result"`
}

type buyingPowerResponse struct {
	Result struct {
		Currency        string `json:"currency"`
		CashBuyingPower string `json:"cashBuyingPower"`
	} `json:"result"`
}

type exchangeRateResponse struct {
	Result struct {
		BaseCurrency  string `json:"baseCurrency"`
		QuoteCurrency string `json:"quoteCurrency"`
		Rate          string `json:"rate"`
	} `json:"result"`
}

type orderCreateResponse struct {
	Result struct {
		OrderID       string  `json:"orderId"`
		ClientOrderID *string `json:"clientOrderId"`
	} `json:"result"`
}

type holdingItem struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Quantity string `json:"quantity"`
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
	token, err := c.accessToken()
	if err != nil {
		return models.KisAccountSnapshot{}, err
	}
	cashBalance, err := c.fetchCashBalanceKRW(token, accountSeq)
	if err != nil {
		return models.KisAccountSnapshot{}, err
	}

	holdings, err := c.fetchHoldings(token, accountSeq)
	if err != nil {
		return models.KisAccountSnapshot{}, err
	}
	return models.KisAccountSnapshot{
		CashBalance: cashBalance,
		Holdings:    holdings,
	}, nil
}

// PlaceOrder creates a market quantity order for the given Toss accountSeq.
func (c *Client) PlaceOrder(accountSeq string, intent models.OrderIntent) (map[string]any, error) {
	accountSeq = strings.TrimSpace(accountSeq)
	if accountSeq == "" {
		return nil, fmt.Errorf("toss order: accountSeq is required")
	}
	if intent.Quantity <= 0 {
		return nil, fmt.Errorf("toss order: quantity must be positive")
	}

	side, err := tossOrderSide(intent.Side)
	if err != nil {
		return nil, err
	}

	token, err := c.accessToken()
	if err != nil {
		return nil, err
	}

	payload := map[string]string{
		"symbol":    strings.ToUpper(strings.TrimSpace(intent.Ticker)),
		"side":      side,
		"orderType": "MARKET",
		"quantity":  fmt.Sprintf("%d", intent.Quantity),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("toss order: json marshal: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v1/orders", bytes.NewReader(body)) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return nil, fmt.Errorf("toss order: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tossinvest-Account", accountSeq)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return nil, fmt.Errorf("toss order: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("toss order: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, parseAPIError("toss order", resp.StatusCode, respBody)
	}

	var parsed orderCreateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("toss order: json unmarshal: %w", err)
	}
	if parsed.Result.OrderID == "" {
		return nil, fmt.Errorf("toss order: missing orderId")
	}

	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("toss order: json unmarshal raw: %w", err)
	}
	return raw, nil
}

func (c *Client) fetchHoldings(token, accountSeq string) ([]models.KisHoldingPosition, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/holdings", nil) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return nil, fmt.Errorf("toss holdings: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tossinvest-Account", accountSeq)

	resp, err := c.HTTP.Do(req) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return nil, fmt.Errorf("toss holdings: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("toss holdings: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, parseAPIError("toss holdings", resp.StatusCode, body)
	}

	var parsed holdingsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("toss holdings: json unmarshal: %w", err)
	}

	bySymbol := map[string]models.KisHoldingPosition{}
	for _, item := range parsed.Result.Items {
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
	return holdings, nil
}

func (c *Client) fetchCashBalanceKRW(token, accountSeq string) (numeric.Decimal, error) {
	krw, err := c.fetchBuyingPower(token, accountSeq, "KRW")
	if err != nil {
		return numeric.Decimal{}, err
	}
	usd, err := c.fetchBuyingPower(token, accountSeq, "USD")
	if err != nil {
		return numeric.Decimal{}, err
	}
	if usd.IsZero() {
		return krw, nil
	}
	rate, err := c.fetchUSDKRWRate(token)
	if err != nil {
		return numeric.Decimal{}, err
	}
	return numeric.Wrap(krw.Decimal.Add(usd.Decimal.Mul(rate.Decimal))), nil
}

func (c *Client) fetchBuyingPower(token, accountSeq, currency string) (numeric.Decimal, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/buying-power", nil) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return numeric.Decimal{}, fmt.Errorf("toss buying-power: create request: %w", err)
	}
	q := req.URL.Query()
	q.Set("currency", currency)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tossinvest-Account", accountSeq)

	resp, err := c.HTTP.Do(req) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return numeric.Decimal{}, fmt.Errorf("toss buying-power: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return numeric.Decimal{}, fmt.Errorf("toss buying-power: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return numeric.Decimal{}, parseAPIError("toss buying-power", resp.StatusCode, body)
	}

	var parsed buyingPowerResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return numeric.Decimal{}, fmt.Errorf("toss buying-power: json unmarshal: %w", err)
	}
	if !strings.EqualFold(parsed.Result.Currency, currency) {
		return numeric.Decimal{}, fmt.Errorf("toss buying-power: unexpected currency %q", parsed.Result.Currency)
	}
	power := numeric.Wrap(parseDecimal(parsed.Result.CashBuyingPower))
	if power.IsNegative() {
		return numeric.Decimal{}, fmt.Errorf("toss buying-power: negative %s cashBuyingPower %q", currency, parsed.Result.CashBuyingPower)
	}
	return power, nil
}

func (c *Client) fetchUSDKRWRate(token string) (numeric.Decimal, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/exchange-rate", nil) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return numeric.Decimal{}, fmt.Errorf("toss exchange-rate: create request: %w", err)
	}
	q := req.URL.Query()
	q.Set("baseCurrency", "USD")
	q.Set("quoteCurrency", "KRW")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTP.Do(req) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return numeric.Decimal{}, fmt.Errorf("toss exchange-rate: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return numeric.Decimal{}, fmt.Errorf("toss exchange-rate: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return numeric.Decimal{}, parseAPIError("toss exchange-rate", resp.StatusCode, body)
	}

	var parsed exchangeRateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return numeric.Decimal{}, fmt.Errorf("toss exchange-rate: json unmarshal: %w", err)
	}
	if !strings.EqualFold(parsed.Result.BaseCurrency, "USD") || !strings.EqualFold(parsed.Result.QuoteCurrency, "KRW") {
		return numeric.Decimal{}, fmt.Errorf("toss exchange-rate: unexpected pair %q/%q",
			parsed.Result.BaseCurrency, parsed.Result.QuoteCurrency)
	}
	rate := parseDecimal(parsed.Result.Rate)
	if !rate.IsPositive() {
		return numeric.Decimal{}, fmt.Errorf("toss exchange-rate: invalid USD/KRW rate %q", parsed.Result.Rate)
	}
	return numeric.Wrap(rate), nil
}

func (c *Client) accessToken() (string, error) {
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
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/oauth2/token", strings.NewReader(form.Encode())) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
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

func parseOAuthError(status int, body []byte) error {
	var data struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &data); err == nil && data.Error != "" {
		if data.ErrorDescription != "" {
			return fmt.Errorf("toss auth HTTP %d: %s: %s", status, data.Error, data.ErrorDescription)
		}
		return fmt.Errorf("toss auth HTTP %d: %s", status, data.Error)
	}
	return fmt.Errorf("toss auth HTTP %d: %s", status, string(body))
}

func parseAPIError(prefix string, status int, body []byte) error {
	var data apiErrorResponse
	if err := json.Unmarshal(body, &data); err == nil && data.Error.Code != "" {
		if data.Error.RequestID != "" {
			return fmt.Errorf("%s HTTP %d: %s: %s (request_id=%s)",
				prefix, status, data.Error.Code, data.Error.Message, data.Error.RequestID)
		}
		return fmt.Errorf("%s HTTP %d: %s: %s", prefix, status, data.Error.Code, data.Error.Message)
	}
	return fmt.Errorf("%s HTTP %d: %s", prefix, status, string(body))
}

func parseDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return decimal.Zero
	}
	return d
}

func tossOrderSide(side string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "buy":
		return "BUY", nil
	case "sell":
		return "SELL", nil
	default:
		return "", fmt.Errorf("toss order: unsupported side %q", side)
	}
}
