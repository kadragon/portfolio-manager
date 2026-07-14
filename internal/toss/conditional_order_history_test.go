package toss

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestGetConditionalOrdersRequiresStatus(t *testing.T) {
	client := NewClient(nil, "http://localhost", "cid", "secret")
	for _, status := range []string{"", "BOGUS", "open"} {
		if _, err := client.GetConditionalOrders(context.Background(), "7", ConditionalOrderListParams{Status: status}); err == nil {
			t.Fatalf("status %q: expected error, got nil", status)
		}
	}
}

func TestGetConditionalOrdersQueryFilters(t *testing.T) {
	var gotQuery, gotAccount string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/conditional-orders" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		gotAccount = r.Header.Get("X-Tossinvest-Account")
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"conditionalOrders": []any{},
			"nextCursor":        nil,
			"hasNext":           false,
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetConditionalOrders(context.Background(), "7", ConditionalOrderListParams{
		Status: "OPEN",
		Symbol: "005930",
		Cursor: "cur-1",
		Limit:  50,
	})
	if err != nil {
		t.Fatalf("GetConditionalOrders: %v", err)
	}
	if len(got.ConditionalOrders) != 0 || got.NextCursor != nil || got.HasNext {
		t.Fatalf("got = %+v", got)
	}
	if gotAccount != "7" {
		t.Fatalf("account header = %q, want 7", gotAccount)
	}
	q, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if q.Get("status") != "OPEN" || q.Get("symbol") != "005930" || q.Get("cursor") != "cur-1" || q.Get("limit") != "50" {
		t.Fatalf("query = %q", gotQuery)
	}
}

func TestGetConditionalOrdersOmitsOptionalFilters(t *testing.T) {
	var gotQuery string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"conditionalOrders": []any{},
			"nextCursor":        nil,
			"hasNext":           false,
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	if _, err := client.GetConditionalOrders(context.Background(), "7", ConditionalOrderListParams{Status: "CLOSED"}); err != nil {
		t.Fatalf("GetConditionalOrders: %v", err)
	}
	q, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if q.Get("status") != "CLOSED" {
		t.Fatalf("query = %q, want status=CLOSED", gotQuery)
	}
	if q.Has("symbol") || q.Has("cursor") || q.Has("limit") {
		t.Fatalf("query = %q, expected optional filters omitted", gotQuery)
	}
}

func TestGetConditionalOrdersHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid-request","message":"유효하지 않은 주문 상태 필터입니다."}}`))
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetConditionalOrders(context.Background(), "7", ConditionalOrderListParams{Status: "OPEN"})
	if err == nil || !strings.Contains(err.Error(), "invalid-request") {
		t.Fatalf("expected invalid-request error, got %v", err)
	}
}

func TestGetConditionalOrderHappyPath(t *testing.T) {
	var gotPath, gotAccount string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAccount = r.Header.Get("X-Tossinvest-Account")
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"conditionalOrderId": "gaZIG-dYMWil8AAXyPmlRg",
			"type":               "SINGLE",
			"status":             "WATCHING",
			"symbol":             "005930",
			"market":             "KR",
			"quantity":           "100",
			"orderType":          "LIMIT",
			"expireDate":         "2026-09-10",
			"first": map[string]any{
				"type":             "STOP",
				"status":           "WATCHING",
				"triggerPrice":     "295",
				"targetProfitRate": nil,
				"orderPrice":       "295",
				"triggeredOrderId": nil,
			},
			"createdAt": "2026-06-12T09:00:00+09:00",
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetConditionalOrder(context.Background(), "7", "gaZIG-dYMWil8AAXyPmlRg")
	if err != nil {
		t.Fatalf("GetConditionalOrder: %v", err)
	}
	if gotPath != "/api/v1/conditional-orders/gaZIG-dYMWil8AAXyPmlRg" {
		t.Fatalf("path = %s", gotPath)
	}
	if gotAccount != "7" {
		t.Fatalf("account header = %q, want 7", gotAccount)
	}
	if got.ConditionalOrderID != "gaZIG-dYMWil8AAXyPmlRg" || got.Type != "SINGLE" || got.Second != nil {
		t.Fatalf("got = %+v", got)
	}
	if got.First.TriggerPrice == nil || *got.First.TriggerPrice != "295" {
		t.Fatalf("first.triggerPrice = %v", got.First.TriggerPrice)
	}
}

func TestGetConditionalOrderPathEscaping(t *testing.T) {
	var gotPath string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"conditionalOrderId": "cond/with-slash",
			"type":               "SINGLE",
			"status":             "WATCHING",
			"symbol":             "005930",
			"market":             "KR",
			"quantity":           "1",
			"orderType":          "LIMIT",
			"expireDate":         "2026-09-10",
			"first": map[string]any{
				"type":   "STOP",
				"status": "WATCHING",
			},
			"createdAt": "2026-06-12T09:00:00+09:00",
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	if _, err := client.GetConditionalOrder(context.Background(), "7", "cond/with-slash"); err != nil {
		t.Fatalf("GetConditionalOrder: %v", err)
	}
	if gotPath != "/api/v1/conditional-orders/cond%2Fwith-slash" {
		t.Fatalf("path = %s", gotPath)
	}
}

func TestGetConditionalOrderHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"conditional-order-not-found","message":"조건주문을 찾을 수 없습니다."}}`))
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetConditionalOrder(context.Background(), "7", "cond-1")
	if err == nil || !strings.Contains(err.Error(), "conditional-order-not-found") {
		t.Fatalf("expected conditional-order-not-found error, got %v", err)
	}
}
