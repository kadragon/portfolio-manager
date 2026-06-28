package toss

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kadragon/portfolio-manager/internal/numeric"
)

func TestFetchAccountSnapshotCombinesKRWAndUSDBuyingPower(t *testing.T) {
	var calls []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.RequestURI())
		switch r.URL.Path {
		case "/oauth2/token":
			if r.Method != http.MethodPost {
				t.Fatalf("token method = %s", r.Method)
			}
			if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
				t.Fatalf("token content-type = %q", got)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("client_id") != "cid" || r.Form.Get("client_secret") != "secret" {
				t.Fatalf("unexpected token form: %v", r.Form)
			}
			writeJSON(t, w, map[string]any{
				"access_token": "token-1",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case "/api/v1/buying-power":
			if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
				t.Fatalf("authorization = %q", got)
			}
			if got := r.Header.Get("X-Tossinvest-Account"); got != "7" {
				t.Fatalf("account header = %q", got)
			}
			currency := r.URL.Query().Get("currency")
			switch currency {
			case "KRW":
				writeJSON(t, w, map[string]any{"result": map[string]any{
					"currency":        "KRW",
					"cashBuyingPower": "1000",
				}})
			case "USD":
				writeJSON(t, w, map[string]any{"result": map[string]any{
					"currency":        "USD",
					"cashBuyingPower": "2.5",
				}})
			default:
				t.Fatalf("unexpected currency %q", currency)
			}
		case "/api/v1/exchange-rate":
			if r.URL.Query().Get("baseCurrency") != "USD" || r.URL.Query().Get("quoteCurrency") != "KRW" {
				t.Fatalf("unexpected exchange-rate query: %s", r.URL.RawQuery)
			}
			writeJSON(t, w, map[string]any{"result": map[string]any{
				"baseCurrency":  "USD",
				"quoteCurrency": "KRW",
				"rate":          "1400",
			}})
		case "/api/v1/holdings":
			writeJSON(t, w, map[string]any{"result": map[string]any{"items": []map[string]any{
				{"symbol": "aapl", "name": "Apple Inc.", "quantity": "1.5"},
				{"symbol": "005930", "name": "삼성전자", "quantity": "3"},
			}}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.Client(), server.URL, "cid", "secret")
	got, err := client.FetchAccountSnapshot("7", "")
	if err != nil {
		t.Fatalf("FetchAccountSnapshot: %v", err)
	}

	if !got.CashBalance.Equal(mustDecimal("4500").Decimal) {
		t.Fatalf("cash = %s, want 4500", got.CashBalance.String())
	}
	if len(got.Holdings) != 2 {
		t.Fatalf("holdings = %+v", got.Holdings)
	}
	if got.Holdings[0].Ticker != "005930" || got.Holdings[1].Ticker != "AAPL" {
		t.Fatalf("holdings order = %+v", got.Holdings)
	}

	wantCalls := []string{
		"POST /oauth2/token",
		"GET /api/v1/buying-power?currency=KRW",
		"GET /api/v1/buying-power?currency=USD",
		"GET /api/v1/exchange-rate?baseCurrency=USD&quoteCurrency=KRW",
		"GET /api/v1/holdings",
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", calls, wantCalls)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode json: %v", err)
	}
}

func mustDecimal(s string) numeric.Decimal {
	d, err := numeric.FromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

// tokenOnlyServer returns a test server that answers /oauth2/token and delegates
// everything else to handler. Reduces boilerplate in error-path tests.
func tokenOnlyServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			writeJSON(t, w, map[string]any{
				"access_token": "tok",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
			return
		}
		handler(w, r)
	}))
}

func TestFetchAccountSnapshotEmptyAccountSeq(t *testing.T) {
	c := NewClient(nil, "http://localhost", "cid", "secret")
	_, err := c.FetchAccountSnapshot("", "")
	if err == nil || !strings.Contains(err.Error(), "accountSeq") {
		t.Fatalf("expected accountSeq error, got %v", err)
	}
}

func TestFetchAccountSnapshotUSDZero(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/buying-power":
			currency := r.URL.Query().Get("currency")
			power := "0"
			if currency == "KRW" {
				power = "500"
			}
			writeJSON(t, w, map[string]any{"result": map[string]any{
				"currency":        currency,
				"cashBuyingPower": power,
			}})
		case "/api/v1/holdings":
			writeJSON(t, w, map[string]any{"result": map[string]any{"items": []any{}}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.FetchAccountSnapshot("1", "")
	if err != nil {
		t.Fatalf("FetchAccountSnapshot: %v", err)
	}
	if !got.CashBalance.Equal(mustDecimal("500").Decimal) {
		t.Fatalf("cash = %s, want 500", got.CashBalance.String())
	}
}

func TestAccessTokenCached(t *testing.T) {
	var tokenCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			tokenCalls++
			writeJSON(t, w, map[string]any{
				"access_token": "tok",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
			return
		}
		if r.URL.Path == "/api/v1/buying-power" {
			writeJSON(t, w, map[string]any{"result": map[string]any{
				"currency":        r.URL.Query().Get("currency"),
				"cashBuyingPower": "0",
			}})
			return
		}
		if r.URL.Path == "/api/v1/holdings" {
			writeJSON(t, w, map[string]any{"result": map[string]any{"items": []any{}}})
			return
		}
	}))
	t.Cleanup(srv.Close)

	now := time.Now()
	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	client.Now = func() time.Time { return now }

	if _, err := client.FetchAccountSnapshot("1", ""); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := client.FetchAccountSnapshot("1", ""); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if tokenCalls != 1 {
		t.Fatalf("token fetched %d times, want 1", tokenCalls)
	}
}

func TestAuthHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"invalid_client","error_description":"bad credentials"}`)
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.FetchAccountSnapshot("1", "")
	if err == nil || !strings.Contains(err.Error(), "invalid_client") {
		t.Fatalf("expected auth error with error code, got %v", err)
	}
}

func TestAuthHTTPErrorBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal server error")
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.FetchAccountSnapshot("1", "")
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected HTTP 500 error, got %v", err)
	}
}

func TestHoldingsHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/buying-power":
			writeJSON(t, w, map[string]any{"result": map[string]any{
				"currency": r.URL.Query().Get("currency"), "cashBuyingPower": "0",
			}})
		case "/api/v1/holdings":
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"code":"INVALID_ACCOUNT","message":"account not found","requestId":"req-1"}}`)
		}
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.FetchAccountSnapshot("1", "")
	if err == nil || !strings.Contains(err.Error(), "INVALID_ACCOUNT") {
		t.Fatalf("expected INVALID_ACCOUNT error, got %v", err)
	}
}

func TestBuyingPowerHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/buying-power" {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"error":{"code":"FORBIDDEN","message":"no access"}}`)
		}
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.FetchAccountSnapshot("1", "")
	if err == nil || !strings.Contains(err.Error(), "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN error, got %v", err)
	}
}

func TestExchangeRateHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/buying-power":
			currency := r.URL.Query().Get("currency")
			power := "100"
			if currency == "KRW" {
				power = "0"
			}
			writeJSON(t, w, map[string]any{"result": map[string]any{
				"currency": currency, "cashBuyingPower": power,
			}})
		case "/api/v1/exchange-rate":
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, "service unavailable")
		}
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.FetchAccountSnapshot("1", "")
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected exchange-rate error, got %v", err)
	}
}

func TestExchangeRateInvalidPair(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/buying-power":
			currency := r.URL.Query().Get("currency")
			power := "100"
			if currency == "KRW" {
				power = "0"
			}
			writeJSON(t, w, map[string]any{"result": map[string]any{
				"currency": currency, "cashBuyingPower": power,
			}})
		case "/api/v1/exchange-rate":
			writeJSON(t, w, map[string]any{"result": map[string]any{
				"baseCurrency": "EUR", "quoteCurrency": "USD", "rate": "1.1",
			}})
		}
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.FetchAccountSnapshot("1", "")
	if err == nil || !strings.Contains(err.Error(), "unexpected pair") {
		t.Fatalf("expected pair error, got %v", err)
	}
}

func TestExchangeRateZeroRate(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/buying-power":
			currency := r.URL.Query().Get("currency")
			power := "100"
			if currency == "KRW" {
				power = "0"
			}
			writeJSON(t, w, map[string]any{"result": map[string]any{
				"currency": currency, "cashBuyingPower": power,
			}})
		case "/api/v1/exchange-rate":
			writeJSON(t, w, map[string]any{"result": map[string]any{
				"baseCurrency": "USD", "quoteCurrency": "KRW", "rate": "0",
			}})
		}
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.FetchAccountSnapshot("1", "")
	if err == nil || !strings.Contains(err.Error(), "invalid USD/KRW rate") {
		t.Fatalf("expected invalid rate error, got %v", err)
	}
}

func TestBuyingPowerUnexpectedCurrency(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/buying-power" {
			writeJSON(t, w, map[string]any{"result": map[string]any{
				"currency": "JPY", "cashBuyingPower": "100",
			}})
		}
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.FetchAccountSnapshot("1", "")
	if err == nil || !strings.Contains(err.Error(), "unexpected currency") {
		t.Fatalf("expected currency error, got %v", err)
	}
}

func TestHoldingsEmptyAndZeroQuantityFiltered(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/buying-power":
			writeJSON(t, w, map[string]any{"result": map[string]any{
				"currency": r.URL.Query().Get("currency"), "cashBuyingPower": "0",
			}})
		case "/api/v1/holdings":
			writeJSON(t, w, map[string]any{"result": map[string]any{"items": []map[string]any{
				{"symbol": "", "name": "no-symbol", "quantity": "1"},
				{"symbol": "AAPL", "name": "Apple", "quantity": "0"},
				{"symbol": "MSFT", "name": "Microsoft", "quantity": "2"},
			}}})
		}
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.FetchAccountSnapshot("1", "")
	if err != nil {
		t.Fatalf("FetchAccountSnapshot: %v", err)
	}
	if len(got.Holdings) != 1 || got.Holdings[0].Ticker != "MSFT" {
		t.Fatalf("holdings = %+v, want [MSFT]", got.Holdings)
	}
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient(nil, "", "cid", "secret")
	if c.HTTP == nil {
		t.Fatal("HTTP client should default to non-nil")
	}
	if c.BaseURL != defaultBaseURL {
		t.Fatalf("BaseURL = %q, want %q", c.BaseURL, defaultBaseURL)
	}
}
