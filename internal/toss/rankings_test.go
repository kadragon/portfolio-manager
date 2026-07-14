package toss

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestGetRankingsHappyPath(t *testing.T) {
	var gotQuery string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/rankings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"rankedAt": "2026-06-10T14:30:00+09:00",
			"rankings": []map[string]any{
				{
					"rank":     1,
					"symbol":   "005930",
					"currency": "KRW",
					"price": map[string]any{
						"lastPrice":  "56500",
						"basePrice":  "55800",
						"changeRate": "0.0125",
					},
					"tradingVolume": "18432100",
					"tradingAmount": "1041436650000",
				},
			},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetRankings(context.Background(), RankingParams{
		Type:                     "MARKET_TRADING_AMOUNT",
		MarketCountry:            "KR",
		Duration:                 "realtime",
		ExcludeInvestmentCaution: true,
		Count:                    50,
	})
	if err != nil {
		t.Fatalf("GetRankings: %v", err)
	}

	q := mustParseQuery(t, gotQuery)
	if q.Get("type") != "MARKET_TRADING_AMOUNT" || q.Get("marketCountry") != "KR" || q.Get("duration") != "realtime" {
		t.Fatalf("query = %q", gotQuery)
	}
	if q.Get("excludeInvestmentCaution") != "true" {
		t.Fatalf("excludeInvestmentCaution missing: %q", gotQuery)
	}
	if q.Get("count") != "50" {
		t.Fatalf("count = %q, want 50", q.Get("count"))
	}

	if got.RankedAt == nil {
		t.Fatal("RankedAt is nil")
	}
	if len(got.Rankings) != 1 {
		t.Fatalf("rankings = %+v", got.Rankings)
	}
	item := got.Rankings[0]
	if item.Rank != 1 || item.Symbol != "005930" || item.Currency != "KRW" {
		t.Fatalf("item = %+v", item)
	}
	if item.Price.ChangeRate == nil || *item.Price.ChangeRate != "0.0125" {
		t.Fatalf("changeRate = %v", item.Price.ChangeRate)
	}
	if item.TradingVolume != "18432100" || item.TradingAmount != "1041436650000" {
		t.Fatalf("item = %+v", item)
	}
}

func TestGetRankingsOptionalFieldsAbsent(t *testing.T) {
	var gotQuery string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"rankedAt": nil,
			"rankings": []any{},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetRankings(context.Background(), RankingParams{
		Type:          "TOP_GAINERS",
		MarketCountry: "US",
		Duration:      "1d",
	})
	if err != nil {
		t.Fatalf("GetRankings: %v", err)
	}

	q := mustParseQuery(t, gotQuery)
	if _, ok := q["excludeInvestmentCaution"]; ok {
		t.Fatalf("excludeInvestmentCaution should be omitted, query = %q", gotQuery)
	}
	if _, ok := q["count"]; ok {
		t.Fatalf("count should be omitted, query = %q", gotQuery)
	}
	if got.RankedAt != nil {
		t.Fatalf("RankedAt = %v, want nil", got.RankedAt)
	}
	if len(got.Rankings) != 0 {
		t.Fatalf("rankings = %+v, want empty", got.Rankings)
	}
}

func TestGetRankingsHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"code":"INVALID_PARAMETER","message":"invalid duration"}}`)
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetRankings(context.Background(), RankingParams{
		Type:          "MARKET_TRADING_AMOUNT",
		MarketCountry: "KR",
		Duration:      "bogus",
	})
	if err == nil || !strings.Contains(err.Error(), "INVALID_PARAMETER") {
		t.Fatalf("expected INVALID_PARAMETER error, got %v", err)
	}
}

func mustParseQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	values, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("parse query %q: %v", raw, err)
	}
	return values
}
