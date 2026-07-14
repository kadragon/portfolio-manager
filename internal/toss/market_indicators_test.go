package toss

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGetMarketIndicatorPricesHappyPath(t *testing.T) {
	var gotQuery string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market-indicators/prices" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		writeJSON(t, w, map[string]any{"result": []map[string]any{
			{"symbol": "KOSPI", "timestamp": "2026-06-11T15:30:00+09:00", "lastPrice": "2812.45"},
			{"symbol": "KOSDAQ", "timestamp": "2026-06-11T15:30:00+09:00", "lastPrice": "845.32"},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetMarketIndicatorPrices(context.Background(), []string{"KOSPI", "KOSDAQ"})
	if err != nil {
		t.Fatalf("GetMarketIndicatorPrices: %v", err)
	}

	q := mustParseQuery(t, gotQuery)
	if q.Get("symbols") != "KOSPI,KOSDAQ" {
		t.Fatalf("symbols query = %q", q.Get("symbols"))
	}
	if len(got) != 2 || got[0].Symbol != "KOSPI" || got[0].LastPrice != "2812.45" {
		t.Fatalf("got = %+v", got)
	}
	if got[0].Timestamp == nil {
		t.Fatal("Timestamp is nil")
	}
}

func TestGetMarketIndicatorPricesOptionalTimestampAbsent(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"result": []map[string]any{
			{"symbol": "KR_BOND_10Y", "timestamp": nil, "lastPrice": "3.25"},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetMarketIndicatorPrices(context.Background(), []string{"KR_BOND_10Y"})
	if err != nil {
		t.Fatalf("GetMarketIndicatorPrices: %v", err)
	}
	if len(got) != 1 || got[0].Timestamp != nil {
		t.Fatalf("got = %+v, want nil timestamp", got)
	}
}

func TestGetMarketIndicatorPricesHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"code":"INVALID_SYMBOL","message":"unknown symbol"}}`)
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetMarketIndicatorPrices(context.Background(), []string{"BOGUS"})
	if err == nil || !strings.Contains(err.Error(), "INVALID_SYMBOL") {
		t.Fatalf("expected INVALID_SYMBOL error, got %v", err)
	}
}

func TestGetMarketIndicatorCandlesHappyPath(t *testing.T) {
	var gotPath, gotQuery string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"candles": []map[string]any{
				{
					"timestamp":  "2026-06-11T09:00:00+09:00",
					"openPrice":  "2798.32",
					"highPrice":  "2820.15",
					"lowPrice":   "2790.1",
					"closePrice": "2812.45",
					"volume":     "542000000",
				},
			},
			"nextBefore": "2026-06-10T09:00:00+09:00",
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	before := time.Date(2026, 6, 11, 9, 0, 0, 0, time.FixedZone("KST", 9*3600))
	got, err := client.GetMarketIndicatorCandles(context.Background(), "KOSPI", "1d", 30, before)
	if err != nil {
		t.Fatalf("GetMarketIndicatorCandles: %v", err)
	}

	if gotPath != "/api/v1/market-indicators/KOSPI/candles" {
		t.Fatalf("path = %q", gotPath)
	}
	q := mustParseQuery(t, gotQuery)
	if q.Get("interval") != "1d" || q.Get("count") != "30" {
		t.Fatalf("query = %q", gotQuery)
	}
	if q.Get("before") == "" {
		t.Fatalf("before missing from query = %q", gotQuery)
	}

	if len(got.Candles) != 1 || got.Candles[0].ClosePrice != "2812.45" {
		t.Fatalf("candles = %+v", got.Candles)
	}
	if got.NextBefore == nil {
		t.Fatal("NextBefore is nil")
	}
}

func TestGetMarketIndicatorCandlesOptionalParamsAbsent(t *testing.T) {
	var gotQuery string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"candles":    []any{},
			"nextBefore": nil,
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetMarketIndicatorCandles(context.Background(), "KOSPI", "1m", 0, time.Time{})
	if err != nil {
		t.Fatalf("GetMarketIndicatorCandles: %v", err)
	}

	q := mustParseQuery(t, gotQuery)
	if _, ok := q["count"]; ok {
		t.Fatalf("count should be omitted, query = %q", gotQuery)
	}
	if _, ok := q["before"]; ok {
		t.Fatalf("before should be omitted, query = %q", gotQuery)
	}
	if got.NextBefore != nil {
		t.Fatalf("NextBefore = %v, want nil", got.NextBefore)
	}
	if len(got.Candles) != 0 {
		t.Fatalf("candles = %+v, want empty", got.Candles)
	}
}

func TestGetMarketIndicatorCandlesHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"code":"UNKNOWN_SYMBOL","message":"unknown symbol"}}`)
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetMarketIndicatorCandles(context.Background(), "BOGUS", "1d", 0, time.Time{})
	if err == nil || !strings.Contains(err.Error(), "UNKNOWN_SYMBOL") {
		t.Fatalf("expected UNKNOWN_SYMBOL error, got %v", err)
	}
}

func TestGetMarketIndicatorInvestorTradingHappyPath(t *testing.T) {
	var gotPath, gotQuery string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"nextUntil": "2026-06-09",
			"records": []map[string]any{
				{
					"date":      "2026-06-11",
					"updatedAt": "2026-06-11T18:10:00+09:00",
					"individual": map[string]any{
						"buyAmount": "5200000000000", "sellAmount": "5350000000000",
					},
					"foreigner": map[string]any{
						"buyAmount": "1000000000000", "sellAmount": "900000000000",
					},
					"institution": map[string]any{
						"buyAmount": "2100000000000", "sellAmount": "2180000000000",
						"breakdown": map[string]any{
							"financialInvestment":       map[string]any{"buyAmount": "1", "sellAmount": "2"},
							"insurance":                 map[string]any{"buyAmount": "1", "sellAmount": "2"},
							"trust":                     map[string]any{"buyAmount": "1", "sellAmount": "2"},
							"privateEquityFund":         map[string]any{"buyAmount": "1", "sellAmount": "2"},
							"bank":                      map[string]any{"buyAmount": "1", "sellAmount": "2"},
							"otherFinancialInstitution": map[string]any{"buyAmount": "1", "sellAmount": "2"},
							"pensionFund":               map[string]any{"buyAmount": "1", "sellAmount": "2"},
						},
					},
					"otherCorporation": map[string]any{
						"buyAmount": "100", "sellAmount": "200",
					},
				},
			},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetMarketIndicatorInvestorTrading(context.Background(), "KOSPI", "1d", 5, "2026-06-11")
	if err != nil {
		t.Fatalf("GetMarketIndicatorInvestorTrading: %v", err)
	}

	if gotPath != "/api/v1/market-indicators/KOSPI/investor-trading" {
		t.Fatalf("path = %q", gotPath)
	}
	q := mustParseQuery(t, gotQuery)
	if q.Get("interval") != "1d" || q.Get("count") != "5" || q.Get("until") != "2026-06-11" {
		t.Fatalf("query = %q", gotQuery)
	}

	if got.NextUntil == nil || *got.NextUntil != "2026-06-09" {
		t.Fatalf("NextUntil = %v", got.NextUntil)
	}
	if len(got.Records) != 1 {
		t.Fatalf("records = %+v", got.Records)
	}
	rec := got.Records[0]
	if rec.Date != "2026-06-11" || rec.Individual.BuyAmount != "5200000000000" {
		t.Fatalf("record = %+v", rec)
	}
	if rec.Institution.Breakdown.PensionFund.BuyAmount != "1" {
		t.Fatalf("breakdown = %+v", rec.Institution.Breakdown)
	}
}

func TestGetMarketIndicatorInvestorTradingOptionalParamsAbsent(t *testing.T) {
	var gotQuery string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"nextUntil": nil,
			"records":   []any{},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetMarketIndicatorInvestorTrading(context.Background(), "KOSDAQ", "1w", 0, "")
	if err != nil {
		t.Fatalf("GetMarketIndicatorInvestorTrading: %v", err)
	}

	q := mustParseQuery(t, gotQuery)
	if _, ok := q["count"]; ok {
		t.Fatalf("count should be omitted, query = %q", gotQuery)
	}
	if _, ok := q["until"]; ok {
		t.Fatalf("until should be omitted, query = %q", gotQuery)
	}
	if got.NextUntil != nil {
		t.Fatalf("NextUntil = %v, want nil", got.NextUntil)
	}
	if len(got.Records) != 0 {
		t.Fatalf("records = %+v, want empty", got.Records)
	}
}

func TestGetMarketIndicatorInvestorTradingHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"code":"INTERNAL","message":"boom"}}`)
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetMarketIndicatorInvestorTrading(context.Background(), "KOSPI", "1d", 0, "")
	if err == nil || !strings.Contains(err.Error(), "INTERNAL") {
		t.Fatalf("expected INTERNAL error, got %v", err)
	}
}

func TestGetMarketIndicatorInvestorTradingInvalidSymbol(t *testing.T) {
	requested := false
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		requested = true
		t.Fatal("server should not be called for an invalid symbol")
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetMarketIndicatorInvestorTrading(context.Background(), "AAPL", "1d", 0, "")
	if err == nil || !strings.Contains(err.Error(), "KOSPI or KOSDAQ") {
		t.Fatalf("expected symbol validation error, got %v", err)
	}
	if requested {
		t.Fatal("server should not have been called")
	}
}
