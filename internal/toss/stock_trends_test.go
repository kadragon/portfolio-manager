package toss

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// trendServer serves one stock-trend endpoint, asserting the path and
// capturing the raw query.
func trendServer(t *testing.T, wantPath string, gotQuery *string, result map[string]any) *Client {
	t.Helper()
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != wantPath {
			t.Fatalf("path = %q, want %q", r.URL.EscapedPath(), wantPath)
		}
		*gotQuery = r.URL.RawQuery
		writeJSON(t, w, map[string]any{"result": result})
	})
	t.Cleanup(srv.Close)
	return NewClient(srv.Client(), srv.URL, "cid", "secret")
}

func volume(buy, sell, net string) map[string]any {
	return map[string]any{"buyVolume": buy, "sellVolume": sell, "netBuyVolume": net}
}

func TestGetStockInvestorTradingDecodesProvisionalNulls(t *testing.T) {
	var gotQuery string
	client := trendServer(t, "/api/v1/stocks/005930/investor-trading", &gotQuery, map[string]any{
		"nextUntil": "2026-10-15",
		"records": []map[string]any{{
			"date":             "2026-10-16",
			"updatedAt":        "2026-10-16T15:40:00+09:00",
			"individual":       nil,
			"foreigner":        volume("100", "40", "60"),
			"institution":      map[string]any{"buyVolume": "10", "sellVolume": "30", "netBuyVolume": "-20", "breakdown": nil},
			"otherCorporation": nil,
			"foreignerHolding": map[string]any{"holdingQuantity": "1000", "limitQuantity": "2000", "holdingRate": "0.5089"},
			"cfd":              nil,
		}},
	})

	got, err := client.GetStockInvestorTrading(context.Background(), "005930", 5, "2026-10-16")
	if err != nil {
		t.Fatalf("GetStockInvestorTrading: %v", err)
	}
	q := mustParseQuery(t, gotQuery)
	if q.Get("count") != "5" || q.Get("until") != "2026-10-16" {
		t.Fatalf("query = %q", gotQuery)
	}
	if got.NextUntil == nil || *got.NextUntil != "2026-10-15" || len(got.Records) != 1 {
		t.Fatalf("got = %+v", got)
	}
	rec := got.Records[0]
	if rec.Individual != nil || rec.OtherCorporation != nil || rec.Institution.Breakdown != nil || rec.Cfd != nil {
		t.Fatalf("provisional fields should be nil: %+v", rec)
	}
	if rec.Foreigner.NetBuyVolume != "60" || rec.Institution.NetBuyVolume != "-20" {
		t.Fatalf("volumes = %+v / %+v", rec.Foreigner, rec.Institution)
	}
	if rec.ForeignerHolding == nil || rec.ForeignerHolding.HoldingRate != "0.5089" {
		t.Fatalf("foreignerHolding = %+v", rec.ForeignerHolding)
	}
}

func TestGetStockInvestorTradingDecodesBreakdown(t *testing.T) {
	var gotQuery string
	v := volume("1", "2", "-1")
	client := trendServer(t, "/api/v1/stocks/005930/investor-trading", &gotQuery, map[string]any{
		"nextUntil": nil,
		"records": []map[string]any{{
			"date":       "2026-10-15",
			"updatedAt":  "2026-10-15T18:00:00+09:00",
			"individual": volume("5", "3", "2"),
			"foreigner":  v,
			"institution": map[string]any{"buyVolume": "7", "sellVolume": "14", "netBuyVolume": "-7", "breakdown": map[string]any{
				"financialInvestment": v, "insurance": v, "trust": v, "privateEquityFund": v,
				"bank": v, "otherFinancialInstitution": v, "pensionFund": volume("1", "2", "-1"),
			}},
			"otherCorporation": v,
			"foreignerHolding": nil,
			"cfd":              map[string]any{"buyBalanceQuantity": "10", "buyBalanceRate": "0.0002", "sellBalanceQuantity": "5", "sellBalanceRate": "0.0001"},
		}},
	})

	got, err := client.GetStockInvestorTrading(context.Background(), "005930", 0, "")
	if err != nil {
		t.Fatalf("GetStockInvestorTrading: %v", err)
	}
	if gotQuery != "" {
		t.Fatalf("query = %q, want none", gotQuery)
	}
	rec := got.Records[0]
	if got.NextUntil != nil || rec.Individual == nil || rec.Individual.NetBuyVolume != "2" {
		t.Fatalf("got = %+v", got)
	}
	if rec.Institution.Breakdown == nil || rec.Institution.Breakdown.PensionFund.NetBuyVolume != "-1" {
		t.Fatalf("breakdown = %+v", rec.Institution.Breakdown)
	}
	if rec.Cfd == nil || rec.Cfd.SellBalanceRate != "0.0001" || rec.ForeignerHolding != nil {
		t.Fatalf("cfd/foreignerHolding = %+v / %+v", rec.Cfd, rec.ForeignerHolding)
	}
}

func TestGetStockProgramTrades(t *testing.T) {
	var gotQuery string
	client := trendServer(t, "/api/v1/stocks/005930/program-trades", &gotQuery, map[string]any{
		"nextUntil": nil,
		"records": []map[string]any{{
			"date":         "2026-10-16",
			"arbitrage":    volume("10", "20", "-10"),
			"nonArbitrage": volume("300", "100", "200"),
		}},
	})

	got, err := client.GetStockProgramTrades(context.Background(), "005930", 3, "")
	if err != nil {
		t.Fatalf("GetStockProgramTrades: %v", err)
	}
	if gotQuery != "count=3" {
		t.Fatalf("query = %q", gotQuery)
	}
	if len(got.Records) != 1 || got.Records[0].Arbitrage.NetBuyVolume != "-10" || got.Records[0].NonArbitrage.BuyVolume != "300" {
		t.Fatalf("got = %+v", got)
	}
}

func TestGetStockShortSellingNullableRates(t *testing.T) {
	var gotQuery string
	client := trendServer(t, "/api/v1/stocks/005930/short-selling", &gotQuery, map[string]any{
		"nextUntil": "2026-09-01",
		"records": []map[string]any{
			{"date": "2026-10-16", "updatedAt": "2026-10-16T18:00:00+09:00", "shortSellingVolume": "1200", "shortSellingAmount": "84000000", "shortSellingVolumeRate": "0.01234", "shortSellingAmountRate": "0.0123"},
			{"date": "2026-10-15", "updatedAt": "2026-10-15T18:00:00+09:00", "shortSellingVolume": "0", "shortSellingAmount": "0", "shortSellingVolumeRate": nil, "shortSellingAmountRate": nil},
		},
	})

	got, err := client.GetStockShortSelling(context.Background(), "005930", 0, "")
	if err != nil {
		t.Fatalf("GetStockShortSelling: %v", err)
	}
	if len(got.Records) != 2 {
		t.Fatalf("got = %+v", got)
	}
	if r := got.Records[0]; r.ShortSellingVolumeRate == nil || *r.ShortSellingVolumeRate != "0.01234" || r.ShortSellingAmount != "84000000" {
		t.Fatalf("record[0] = %+v", r)
	}
	if r := got.Records[1]; r.ShortSellingVolumeRate != nil || r.ShortSellingAmountRate != nil {
		t.Fatalf("record[1] rates should be nil: %+v", r)
	}
}

func TestGetStockCreditTradesOneSidedNull(t *testing.T) {
	var gotQuery string
	client := trendServer(t, "/api/v1/stocks/005930/credit-trades", &gotQuery, map[string]any{
		"nextUntil": nil,
		"records": []map[string]any{{
			"date":       "2026-10-15",
			"updatedAt":  "2026-10-16T06:00:00+09:00",
			"marginLoan": map[string]any{"newQuantity": "100", "returnQuantity": "50", "balanceQuantity": "5000", "balanceRate": "0.0042", "tradingRate": "0.09"},
			"stockLoan":  nil,
		}},
	})

	got, err := client.GetStockCreditTrades(context.Background(), "005930", 0, "")
	if err != nil {
		t.Fatalf("GetStockCreditTrades: %v", err)
	}
	r := got.Records[0]
	if r.MarginLoan == nil || r.MarginLoan.BalanceRate != "0.0042" || r.MarginLoan.TradingRate != "0.09" || r.StockLoan != nil {
		t.Fatalf("record = %+v", r)
	}
}

func TestGetStockSecuritiesLending(t *testing.T) {
	var gotQuery string
	client := trendServer(t, "/api/v1/stocks/0101N0/securities-lending", &gotQuery, map[string]any{
		"nextUntil": nil,
		"records": []map[string]any{{
			"date": "2026-10-16", "updatedAt": "2026-10-16T18:00:00+09:00",
			"executionQuantity": "10", "repaymentQuantity": "20", "balanceQuantity": "3000", "balanceAmount": "210000000",
		}},
	})

	got, err := client.GetStockSecuritiesLending(context.Background(), "0101N0", 0, "")
	if err != nil {
		t.Fatalf("GetStockSecuritiesLending: %v", err)
	}
	r := got.Records[0]
	if r.ExecutionQuantity != "10" || r.RepaymentQuantity != "20" || r.BalanceQuantity != "3000" || r.BalanceAmount != "210000000" {
		t.Fatalf("record = %+v", r)
	}
}

func TestGetStockTrendEscapesSymbol(t *testing.T) {
	var gotQuery string
	client := trendServer(t, "/api/v1/stocks/BRK%2FB/program-trades", &gotQuery, map[string]any{"nextUntil": nil, "records": []any{}})
	if _, err := client.GetStockProgramTrades(context.Background(), "BRK/B", 0, ""); err != nil {
		t.Fatalf("GetStockProgramTrades: %v", err)
	}
}

func TestGetStockTrendUnsupportedMarket(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"code":"unsupported-market","message":"KR only"}}`)
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetStockShortSelling(context.Background(), "AAPL", 0, "")
	if err == nil || !strings.Contains(err.Error(), "unsupported-market") {
		t.Fatalf("expected unsupported-market error, got %v", err)
	}
}
