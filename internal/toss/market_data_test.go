package toss

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGetOrderbook(t *testing.T) {
	var gotQuery string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/api/v1/orderbook" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotQuery = r.URL.Query().Get("symbol")
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"timestamp": "2026-03-25T09:30:00.123+09:00",
			"currency":  "KRW",
			"asks":      []map[string]any{{"price": "72300", "volume": "1200"}},
			"bids":      []map[string]any{{"price": "72000", "volume": "5200"}},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetOrderbook(context.Background(), "005930")
	if err != nil {
		t.Fatalf("GetOrderbook: %v", err)
	}
	if gotQuery != "005930" {
		t.Fatalf("symbol query = %q, want 005930", gotQuery)
	}
	if got.Currency != "KRW" || len(got.Asks) != 1 || len(got.Bids) != 1 {
		t.Fatalf("got = %+v", got)
	}
	if got.Asks[0].Price != "72300" || got.Bids[0].Volume != "5200" {
		t.Fatalf("entries = %+v", got)
	}
	if got.Timestamp == nil || got.Timestamp.IsZero() {
		t.Fatalf("timestamp = %v, want non-nil", got.Timestamp)
	}
}

func TestGetOrderbookHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"INVALID_SYMBOL","message":"bad symbol","requestId":"req-1"}}`))
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetOrderbook(context.Background(), "bogus")
	if err == nil || !strings.Contains(err.Error(), "INVALID_SYMBOL") {
		t.Fatalf("expected INVALID_SYMBOL error, got %v", err)
	}
}

func TestGetPrices(t *testing.T) {
	var gotQuery string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/prices" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotQuery = r.URL.Query().Get("symbols")
		writeJSON(t, w, map[string]any{"result": []map[string]any{
			{"symbol": "005930", "timestamp": "2026-03-25T09:30:00.123+09:00", "lastPrice": "72000", "currency": "KRW"},
			{"symbol": "AAPL", "timestamp": nil, "lastPrice": "185.70", "currency": "USD"},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetPrices(context.Background(), []string{"005930", "AAPL"})
	if err != nil {
		t.Fatalf("GetPrices: %v", err)
	}
	if gotQuery != "005930,AAPL" {
		t.Fatalf("symbols query = %q, want 005930,AAPL", gotQuery)
	}
	if len(got) != 2 {
		t.Fatalf("got = %+v", got)
	}
	if got[0].Symbol != "005930" || got[0].LastPrice != "72000" || got[0].Timestamp == nil {
		t.Fatalf("got[0] = %+v", got[0])
	}
	if got[1].Symbol != "AAPL" || got[1].Timestamp != nil {
		t.Fatalf("got[1] = %+v, want nil timestamp", got[1])
	}
}

func TestGetPricesHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"INVALID_BATCH_SIZE","message":"too many symbols","requestId":"req-2"}}`))
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetPrices(context.Background(), []string{"005930"})
	if err == nil || !strings.Contains(err.Error(), "INVALID_BATCH_SIZE") {
		t.Fatalf("expected INVALID_BATCH_SIZE error, got %v", err)
	}
}

func TestGetTrades(t *testing.T) {
	var gotSymbol, gotCount string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/trades" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotSymbol = r.URL.Query().Get("symbol")
		gotCount = r.URL.Query().Get("count")
		writeJSON(t, w, map[string]any{"result": []map[string]any{
			{"price": "72000", "volume": "120", "timestamp": "2026-03-25T09:30:42.000+09:00", "currency": "KRW"},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetTrades(context.Background(), "005930", 10)
	if err != nil {
		t.Fatalf("GetTrades: %v", err)
	}
	if gotSymbol != "005930" || gotCount != "10" {
		t.Fatalf("query = symbol=%q count=%q", gotSymbol, gotCount)
	}
	if len(got) != 1 || got[0].Price != "72000" || got[0].Volume != "120" {
		t.Fatalf("got = %+v", got)
	}
}

func TestGetTradesOmitsCountWhenNonPositive(t *testing.T) {
	var sawCount bool
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, sawCount = r.URL.Query()["count"]
		writeJSON(t, w, map[string]any{"result": []map[string]any{}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	if _, err := client.GetTrades(context.Background(), "005930", 0); err != nil {
		t.Fatalf("GetTrades: %v", err)
	}
	if sawCount {
		t.Fatal("count query param should be omitted when count <= 0")
	}
}

func TestGetTradesHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"RATE_LIMIT_EXCEEDED","message":"slow down","requestId":"req-3"}}`))
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetTrades(context.Background(), "005930", 5)
	if err == nil || !strings.Contains(err.Error(), "RATE_LIMIT_EXCEEDED") {
		t.Fatalf("expected RATE_LIMIT_EXCEEDED error, got %v", err)
	}
}

func TestGetPriceLimit(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/price-limits" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("symbol") != "005930" {
			t.Fatalf("symbol query = %q", r.URL.Query().Get("symbol"))
		}
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"timestamp":       "2026-03-25T09:30:00.123+09:00",
			"upperLimitPrice": "93000",
			"lowerLimitPrice": "50400",
			"currency":        "KRW",
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetPriceLimit(context.Background(), "005930")
	if err != nil {
		t.Fatalf("GetPriceLimit: %v", err)
	}
	if got.UpperLimitPrice == nil || *got.UpperLimitPrice != "93000" {
		t.Fatalf("upperLimitPrice = %+v", got.UpperLimitPrice)
	}
	if got.LowerLimitPrice == nil || *got.LowerLimitPrice != "50400" {
		t.Fatalf("lowerLimitPrice = %+v", got.LowerLimitPrice)
	}
}

func TestGetPriceLimitNullLimitsForUSMarket(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"timestamp":       "2026-03-25T22:30:00.456+09:00",
			"upperLimitPrice": nil,
			"lowerLimitPrice": nil,
			"currency":        "USD",
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetPriceLimit(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("GetPriceLimit: %v", err)
	}
	if got.UpperLimitPrice != nil || got.LowerLimitPrice != nil {
		t.Fatalf("got = %+v, want nil limits", got)
	}
}

func TestGetPriceLimitHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"symbol not found","requestId":"req-4"}}`))
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetPriceLimit(context.Background(), "BOGUS")
	if err == nil || !strings.Contains(err.Error(), "NOT_FOUND") {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestGetCandles(t *testing.T) {
	var gotQuery map[string]string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/candles" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		q := r.URL.Query()
		gotQuery = map[string]string{
			"symbol":   q.Get("symbol"),
			"interval": q.Get("interval"),
			"count":    q.Get("count"),
			"before":   q.Get("before"),
			"adjusted": q.Get("adjusted"),
		}
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"candles": []map[string]any{
				{
					"timestamp":  "2026-03-25T09:00:00+09:00",
					"openPrice":  "71600",
					"highPrice":  "72300",
					"lowPrice":   "71500",
					"closePrice": "72000",
					"volume":     "3521000",
					"currency":   "KRW",
				},
			},
			"nextBefore": nil,
		}})
	})
	t.Cleanup(srv.Close)

	before := time.Date(2026, 3, 25, 9, 0, 0, 0, time.FixedZone("KST", 9*60*60))
	adjusted := true
	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetCandles(context.Background(), "005930", "1d", 50, before, &adjusted)
	if err != nil {
		t.Fatalf("GetCandles: %v", err)
	}
	if gotQuery["symbol"] != "005930" || gotQuery["interval"] != "1d" || gotQuery["count"] != "50" ||
		gotQuery["adjusted"] != "true" || gotQuery["before"] == "" {
		t.Fatalf("query = %+v", gotQuery)
	}
	if len(got.Candles) != 1 || got.Candles[0].ClosePrice != "72000" {
		t.Fatalf("candles = %+v", got.Candles)
	}
	if got.NextBefore != nil {
		t.Fatalf("nextBefore = %v, want nil", got.NextBefore)
	}
}

func TestGetCandlesOmitsOptionalParams(t *testing.T) {
	var seen map[string]bool
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		seen = map[string]bool{
			"count":    q.Has("count"),
			"before":   q.Has("before"),
			"adjusted": q.Has("adjusted"),
		}
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"candles":    []map[string]any{},
			"nextBefore": "2026-03-25T09:00:00+09:00",
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetCandles(context.Background(), "005930", "1m", 0, time.Time{}, nil)
	if err != nil {
		t.Fatalf("GetCandles: %v", err)
	}
	if seen["count"] || seen["before"] || seen["adjusted"] {
		t.Fatalf("expected optional params omitted, got %+v", seen)
	}
	if got.NextBefore == nil || got.NextBefore.IsZero() {
		t.Fatalf("nextBefore = %v, want non-nil", got.NextBefore)
	}
}

func TestGetCandlesHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"unexpected","requestId":"req-5"}}`))
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetCandles(context.Background(), "005930", "1d", 0, time.Time{}, nil)
	if err == nil || !strings.Contains(err.Error(), "INTERNAL_ERROR") {
		t.Fatalf("expected INTERNAL_ERROR error, got %v", err)
	}
}
