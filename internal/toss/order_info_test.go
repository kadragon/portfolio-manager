package toss

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestGetSellableQuantityHappyPath(t *testing.T) {
	var gotAccountHeader, gotSymbol string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sellable-quantity" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotAccountHeader = r.Header.Get("X-Tossinvest-Account")
		gotSymbol = r.URL.Query().Get("symbol")
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"sellableQuantity": "100",
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetSellableQuantity(context.Background(), "7", "AAPL")
	if err != nil {
		t.Fatalf("GetSellableQuantity: %v", err)
	}
	if gotAccountHeader != "7" {
		t.Fatalf("account header = %q, want 7", gotAccountHeader)
	}
	if gotSymbol != "AAPL" {
		t.Fatalf("symbol query = %q, want AAPL", gotSymbol)
	}
	if got.SellableQuantity != "100" {
		t.Fatalf("SellableQuantity = %q, want 100", got.SellableQuantity)
	}
}

func TestGetSellableQuantityHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/sellable-quantity" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"code":"account-not-found","message":"계좌를 찾을 수 없습니다.","requestId":"req-1"}}`)
		}
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetSellableQuantity(context.Background(), "7", "AAPL")
	if err == nil || !strings.Contains(err.Error(), "account-not-found") {
		t.Fatalf("expected account-not-found error, got %v", err)
	}
}

func TestGetCommissionsHappyPath(t *testing.T) {
	var gotAccountHeader string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/commissions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotAccountHeader = r.Header.Get("X-Tossinvest-Account")
		writeJSON(t, w, map[string]any{"result": []map[string]any{
			{
				"marketCountry":  "KR",
				"commissionRate": "0.015",
				"startDate":      "2026-01-01",
				"endDate":        "2026-12-31",
			},
			{
				"marketCountry":  "US",
				"commissionRate": "0.1",
				"startDate":      nil,
				"endDate":        nil,
			},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetCommissions(context.Background(), "7")
	if err != nil {
		t.Fatalf("GetCommissions: %v", err)
	}
	if gotAccountHeader != "7" {
		t.Fatalf("account header = %q, want 7", gotAccountHeader)
	}
	if len(got) != 2 {
		t.Fatalf("commissions = %+v, want 2 entries", got)
	}
	if got[0].MarketCountry != "KR" || got[0].StartDate == nil || *got[0].StartDate != "2026-01-01" {
		t.Fatalf("commissions[0] = %+v", got[0])
	}
	// nullable-field-absent case: US entry has null startDate/endDate.
	if got[1].MarketCountry != "US" || got[1].StartDate != nil || got[1].EndDate != nil {
		t.Fatalf("commissions[1] = %+v, want nil start/end dates", got[1])
	}
}

func TestGetCommissionsHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/commissions" {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"error":{"code":"FORBIDDEN","message":"no access"}}`)
		}
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetCommissions(context.Background(), "7")
	if err == nil || !strings.Contains(err.Error(), "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN error, got %v", err)
	}
}
