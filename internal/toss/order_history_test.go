package toss

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestGetOrdersHappyPathWithPagination(t *testing.T) {
	var gotAccountHeader string
	var gotQuery url.Values
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/orders" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotAccountHeader = r.Header.Get("X-Tossinvest-Account")
		gotQuery = r.URL.Query()
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"orders": []map[string]any{
				{
					"orderId":     "ord-1",
					"symbol":      "005930",
					"side":        "BUY",
					"orderType":   "LIMIT",
					"timeInForce": "DAY",
					"status":      "FILLED",
					"price":       "70000",
					"quantity":    "10",
					"orderAmount": nil,
					"currency":    "KRW",
					"orderedAt":   "2026-03-28T09:30:00+09:00",
					"canceledAt":  nil,
					"execution": map[string]any{
						"filledQuantity":     "10",
						"averageFilledPrice": "70000",
						"filledAmount":       "700000",
						"commission":         "1400",
						"tax":                "0",
						"filledAt":           "2026-03-28T09:31:15+09:00",
						"settlementDate":     "2026-03-30",
					},
				},
				{
					"orderId":     "ord-2",
					"symbol":      "AAPL",
					"side":        "BUY",
					"orderType":   "MARKET",
					"timeInForce": "DAY",
					"status":      "PENDING",
					"price":       nil,
					"quantity":    "5",
					"orderAmount": nil,
					"currency":    "USD",
					"orderedAt":   "2026-03-29T09:30:00+09:00",
					"canceledAt":  nil,
					"execution": map[string]any{
						"filledQuantity":     "0",
						"averageFilledPrice": nil,
						"filledAmount":       nil,
						"commission":         nil,
						"tax":                nil,
						"filledAt":           nil,
						"settlementDate":     nil,
					},
				},
			},
			"nextCursor": nil,
			"hasNext":    false,
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetOrders(context.Background(), "7", OrderListParams{Status: "OPEN"})
	if err != nil {
		t.Fatalf("GetOrders: %v", err)
	}
	if gotAccountHeader != "7" {
		t.Fatalf("account header = %q, want 7", gotAccountHeader)
	}
	if gotQuery.Get("status") != "OPEN" {
		t.Fatalf("status query = %q, want OPEN", gotQuery.Get("status"))
	}
	for _, k := range []string{"symbol", "from", "to", "cursor", "limit"} {
		if gotQuery.Has(k) {
			t.Fatalf("expected %q query param to be omitted, got %q", k, gotQuery.Get(k))
		}
	}
	if len(got.Orders) != 2 {
		t.Fatalf("orders = %+v, want 2 entries", got.Orders)
	}
	if got.Orders[0].OrderID != "ord-1" || got.Orders[0].Status != "FILLED" {
		t.Fatalf("orders[0] = %+v", got.Orders[0])
	}
	if got.Orders[0].Execution.FilledAt == nil {
		t.Fatalf("orders[0].Execution.FilledAt = nil, want non-nil")
	}
	// nullable-field-absent case: pending order has null price/canceledAt and a
	// zero-value (all-null) execution.
	if got.Orders[1].Price != nil {
		t.Fatalf("orders[1].Price = %v, want nil", got.Orders[1].Price)
	}
	if got.Orders[1].CanceledAt != nil {
		t.Fatalf("orders[1].CanceledAt = %v, want nil", got.Orders[1].CanceledAt)
	}
	if got.Orders[1].Execution.FilledAt != nil {
		t.Fatalf("orders[1].Execution.FilledAt = %v, want nil", got.Orders[1].Execution.FilledAt)
	}
	if got.NextCursor != nil {
		t.Fatalf("NextCursor = %v, want nil", got.NextCursor)
	}
	if got.HasNext {
		t.Fatalf("HasNext = true, want false")
	}
}

func TestGetOrdersOptionalFiltersSet(t *testing.T) {
	var gotQuery url.Values
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/orders" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotQuery = r.URL.Query()
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"orders":     []map[string]any{},
			"nextCursor": "cursor-2",
			"hasNext":    true,
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetOrders(context.Background(), "7", OrderListParams{
		Status: "CLOSED",
		Symbol: "AAPL",
		From:   "2026-03-01",
		To:     "2026-03-31",
		Cursor: "cursor-1",
		Limit:  50,
	})
	if err != nil {
		t.Fatalf("GetOrders: %v", err)
	}
	wantQuery := map[string]string{
		"status": "CLOSED",
		"symbol": "AAPL",
		"from":   "2026-03-01",
		"to":     "2026-03-31",
		"cursor": "cursor-1",
		"limit":  "50",
	}
	for k, want := range wantQuery {
		if got := gotQuery.Get(k); got != want {
			t.Fatalf("query %q = %q, want %q", k, got, want)
		}
	}
	if got.NextCursor == nil || *got.NextCursor != "cursor-2" {
		t.Fatalf("NextCursor = %v, want cursor-2", got.NextCursor)
	}
	if !got.HasNext {
		t.Fatalf("HasNext = false, want true")
	}
}

func TestGetOrdersLimitOmittedWhenNonPositive(t *testing.T) {
	var gotQuery url.Values
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"orders": []map[string]any{}, "nextCursor": nil, "hasNext": false,
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	if _, err := client.GetOrders(context.Background(), "7", OrderListParams{Status: "OPEN", Limit: 0}); err != nil {
		t.Fatalf("GetOrders: %v", err)
	}
	if gotQuery.Has("limit") {
		t.Fatalf("expected limit query param omitted, got %q", gotQuery.Get("limit"))
	}
}

func TestGetOrdersClosedStatusHTTPError(t *testing.T) {
	// Per the current API spec, status=CLOSED returns a 400
	// closed-not-supported error; verify parseAPIError surfaces it untouched.
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/orders" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"code":"closed-not-supported","message":"CLOSED status is not yet supported.","requestId":"req-1"}}`)
		}
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetOrders(context.Background(), "7", OrderListParams{Status: "CLOSED"})
	if err == nil || !strings.Contains(err.Error(), "closed-not-supported") {
		t.Fatalf("expected closed-not-supported error, got %v", err)
	}
}

func TestGetOrderHappyPath(t *testing.T) {
	var gotPath, gotAccountHeader string
	orderID := "0d5QIHjmtksbsmM-hBRAgP-ExI8iodGm9fAR5txelPfnMM8XQ_swoJdwL5RpGWMo"
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAccountHeader = r.Header.Get("X-Tossinvest-Account")
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"orderId":     orderID,
			"symbol":      "005930",
			"side":        "BUY",
			"orderType":   "LIMIT",
			"timeInForce": "DAY",
			"status":      "FILLED",
			"price":       "70000",
			"quantity":    "10",
			"orderAmount": nil,
			"currency":    "KRW",
			"orderedAt":   "2026-03-28T09:30:00+09:00",
			"canceledAt":  nil,
			"execution": map[string]any{
				"filledQuantity":     "10",
				"averageFilledPrice": "70000",
				"filledAmount":       "700000",
				"commission":         "1400",
				"tax":                "0",
				"filledAt":           "2026-03-28T09:31:15+09:00",
				"settlementDate":     "2026-03-30",
			},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetOrder(context.Background(), "7", orderID)
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if gotAccountHeader != "7" {
		t.Fatalf("account header = %q, want 7", gotAccountHeader)
	}
	wantPath := "/api/v1/orders/" + url.PathEscape(orderID)
	if gotPath != wantPath {
		t.Fatalf("path = %q, want %q", gotPath, wantPath)
	}
	if got.OrderID != orderID || got.Status != "FILLED" {
		t.Fatalf("order = %+v", got)
	}
}

func TestGetOrderPathParamEscaping(t *testing.T) {
	var gotRequestURI string
	orderID := "abc/def def"
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		// r.URL.Path is the decoded form (Go decodes %2F back to '/'), so
		// assert against the raw wire request target instead.
		gotRequestURI = r.RequestURI
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"orderId":     orderID,
			"symbol":      "AAPL",
			"side":        "BUY",
			"orderType":   "MARKET",
			"timeInForce": "DAY",
			"status":      "PENDING",
			"price":       nil,
			"quantity":    "1",
			"orderAmount": nil,
			"currency":    "USD",
			"orderedAt":   "2026-03-29T09:30:00+09:00",
			"canceledAt":  nil,
			"execution": map[string]any{
				"filledQuantity": "0", "averageFilledPrice": nil, "filledAmount": nil,
				"commission": nil, "tax": nil, "filledAt": nil, "settlementDate": nil,
			},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	if _, err := client.GetOrder(context.Background(), "7", orderID); err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if !strings.Contains(gotRequestURI, "%2F") || !strings.Contains(gotRequestURI, "%20") {
		t.Fatalf("request URI %q was not escaped", gotRequestURI)
	}
	if strings.Contains(gotRequestURI, "abc/def def") {
		t.Fatalf("request URI %q contains unescaped orderId", gotRequestURI)
	}
}

func TestGetOrderHTTPErrorNotFound(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"code":"order-not-found","message":"주문을 찾을 수 없습니다.","requestId":"req-1"}}`)
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetOrder(context.Background(), "7", "missing-order")
	if err == nil || !strings.Contains(err.Error(), "order-not-found") {
		t.Fatalf("expected order-not-found error, got %v", err)
	}
}
