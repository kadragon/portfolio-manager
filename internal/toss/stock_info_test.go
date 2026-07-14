package toss

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetStocksReturnsDecodedInfo(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		writeJSON(t, w, map[string]any{"result": []map[string]any{
			{
				"symbol":            "005930",
				"name":              "삼성전자",
				"englishName":       "SamsungElec",
				"isinCode":          "KR7005930003",
				"market":            "KOSPI",
				"securityType":      "STOCK",
				"isCommonShare":     true,
				"status":            "ACTIVE",
				"currency":          "KRW",
				"listDate":          "1975-06-11",
				"delistDate":        nil,
				"sharesOutstanding": "5919637922",
				"leverageFactor":    nil,
				"koreanMarketDetail": map[string]any{
					"liquidationTrading":  false,
					"nxtSupported":        true,
					"krxTradingSuspended": false,
					"nxtTradingSuspended": false,
				},
			},
			{
				"symbol":             "AAPL",
				"name":               "애플",
				"englishName":        "APPLE INC",
				"isinCode":           "US0378331005",
				"market":             "NASDAQ",
				"securityType":       "STOCK",
				"isCommonShare":      true,
				"status":             "ACTIVE",
				"currency":           "USD",
				"listDate":           "1980-12-12",
				"delistDate":         nil,
				"sharesOutstanding":  "14702703000",
				"leverageFactor":     nil,
				"koreanMarketDetail": nil,
			},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetStocks(context.Background(), []string{"005930", "AAPL"})
	if err != nil {
		t.Fatalf("GetStocks: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/api/v1/stocks" {
		t.Fatalf("path = %s, want /api/v1/stocks", gotPath)
	}
	if gotQuery != "symbols=005930%2CAAPL" {
		t.Fatalf("query = %s, want symbols=005930%%2CAAPL", gotQuery)
	}

	if len(got) != 2 {
		t.Fatalf("got %d stocks, want 2", len(got))
	}
	first := got[0]
	if first.Symbol != "005930" || first.Name != "삼성전자" || first.Market != "KOSPI" ||
		first.SecurityType != "STOCK" || !first.IsCommonShare || first.Status != "ACTIVE" ||
		first.Currency != "KRW" || first.SharesOutstanding != "5919637922" {
		t.Fatalf("first stock = %+v", first)
	}
	if first.ListDate == nil || *first.ListDate != "1975-06-11" {
		t.Fatalf("first.ListDate = %v, want 1975-06-11", first.ListDate)
	}
	if first.DelistDate != nil {
		t.Fatalf("first.DelistDate = %v, want nil", *first.DelistDate)
	}
	if first.LeverageFactor != nil {
		t.Fatalf("first.LeverageFactor = %v, want nil", *first.LeverageFactor)
	}
	if first.KoreanMarketDetail == nil {
		t.Fatal("first.KoreanMarketDetail = nil, want non-nil")
	}
	if !first.KoreanMarketDetail.NxtSupported || first.KoreanMarketDetail.LiquidationTrading ||
		first.KoreanMarketDetail.KrxTradingSuspended {
		t.Fatalf("first.KoreanMarketDetail = %+v", first.KoreanMarketDetail)
	}
	if first.KoreanMarketDetail.NxtTradingSuspended == nil || *first.KoreanMarketDetail.NxtTradingSuspended {
		t.Fatalf("first.KoreanMarketDetail.NxtTradingSuspended = %v, want false", first.KoreanMarketDetail.NxtTradingSuspended)
	}

	second := got[1]
	if second.Symbol != "AAPL" || second.Market != "NASDAQ" || second.Currency != "USD" {
		t.Fatalf("second stock = %+v", second)
	}
	// Overseas listing: koreanMarketDetail is absent/nil — must decode without panicking.
	if second.KoreanMarketDetail != nil {
		t.Fatalf("second.KoreanMarketDetail = %+v, want nil", second.KoreanMarketDetail)
	}
}

func TestGetStocksHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid-request","message":"too many symbols","requestId":"req-1"}}`))
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetStocks(context.Background(), []string{"005930"})
	if err == nil || !strings.Contains(err.Error(), "invalid-request") {
		t.Fatalf("expected invalid-request error, got %v", err)
	}
}

func TestGetStockWarningsReturnsDecodedWarnings(t *testing.T) {
	var gotMethod, gotPath string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		writeJSON(t, w, map[string]any{"result": []map[string]any{
			{
				"warningType": "OVERHEATED",
				"exchange":    "KRX",
				"startDate":   "2026-03-20",
				"endDate":     "2026-03-27",
			},
			{
				"warningType": "VI_STATIC",
				"exchange":    "KRX",
				"startDate":   "2026-03-26",
				"endDate":     nil,
			},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetStockWarnings(context.Background(), "005930")
	if err != nil {
		t.Fatalf("GetStockWarnings: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/api/v1/stocks/005930/warnings" {
		t.Fatalf("path = %s, want /api/v1/stocks/005930/warnings", gotPath)
	}

	if len(got) != 2 {
		t.Fatalf("got %d warnings, want 2", len(got))
	}
	if got[0].WarningType != "OVERHEATED" || got[0].Exchange == nil || *got[0].Exchange != "KRX" ||
		got[0].StartDate == nil || *got[0].StartDate != "2026-03-20" ||
		got[0].EndDate == nil || *got[0].EndDate != "2026-03-27" {
		t.Fatalf("warning[0] = %+v", got[0])
	}
	// endDate null (ongoing) — must decode to nil without panicking.
	if got[1].WarningType != "VI_STATIC" || got[1].EndDate != nil {
		t.Fatalf("warning[1] = %+v, want EndDate nil", got[1])
	}
}

func TestGetStockWarningsEmbedsSymbolInPath(t *testing.T) {
	var gotPath string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(t, w, map[string]any{"result": []map[string]any{}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetStockWarnings(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("GetStockWarnings: %v", err)
	}
	if gotPath != "/api/v1/stocks/AAPL/warnings" {
		t.Fatalf("path = %s, want /api/v1/stocks/AAPL/warnings", gotPath)
	}
	if len(got) != 0 {
		t.Fatalf("got %d warnings, want 0", len(got))
	}
}

func TestGetStockWarningsHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"stock-not-found","message":"no such symbol","requestId":"req-2"}}`))
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetStockWarnings(context.Background(), "NOPE")
	if err == nil || !strings.Contains(err.Error(), "stock-not-found") {
		t.Fatalf("expected stock-not-found error, got %v", err)
	}
}
