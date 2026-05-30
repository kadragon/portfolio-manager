package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// EximClient fetches exchange rates from the Korea EXIM Bank open API.
// Base URL: https://oapi.koreaexim.go.kr
type EximClient struct {
	HTTPClient *http.Client
	BaseURL    string
	AuthKey    string
}

type eximEntry struct {
	CurUnit  string `json:"cur_unit"`
	DealBasR string `json:"deal_bas_r"`
}

// FetchUSDRate returns the USD/KRW deal base rate for searchDate (YYYYMMDD format).
// Returns an error when USD is not present in the response.
func (c *EximClient) FetchUSDRate(searchDate string) (float64, error) {
	req, err := http.NewRequest(http.MethodGet,
		c.BaseURL+"/site/program/financial/exchangeJSON", nil)
	if err != nil {
		return 0, err
	}
	q := req.URL.Query()
	q.Set("authkey", c.AuthKey)
	q.Set("searchdate", searchDate)
	q.Set("data", "AP01")
	req.URL.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("EXIM HTTP %d", resp.StatusCode)
	}

	var entries []eximEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return 0, err
	}
	for _, e := range entries {
		if e.CurUnit == "USD" {
			raw := strings.ReplaceAll(e.DealBasR, ",", "")
			if raw == "" {
				return 0, fmt.Errorf("EXIM: empty deal_bas_r for USD on %s", searchDate)
			}
			var rate float64
			if _, scanErr := fmt.Sscanf(raw, "%f", &rate); scanErr != nil {
				return 0, fmt.Errorf("EXIM: parse %q: %w", raw, scanErr)
			}
			return rate, nil
		}
	}
	return 0, fmt.Errorf("EXIM: USD rate not found for %s", searchDate)
}
