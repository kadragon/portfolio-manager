package toss

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateOrderValidatesBothQuantityAndOrderAmountSet(t *testing.T) {
	client := NewClient(nil, "http://localhost", "cid", "secret")
	_, err := client.CreateOrder(context.Background(), "7", OrderCreateRequest{
		Symbol:      "AAPL",
		Side:        "BUY",
		OrderType:   "MARKET",
		Quantity:    "1",
		OrderAmount: "100",
	})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually-exclusive error, got %v", err)
	}
}

func TestCreateOrderValidatesNeitherQuantityNorOrderAmountSet(t *testing.T) {
	client := NewClient(nil, "http://localhost", "cid", "secret")
	_, err := client.CreateOrder(context.Background(), "7", OrderCreateRequest{
		Symbol:    "AAPL",
		Side:      "BUY",
		OrderType: "MARKET",
	})
	if err == nil || !strings.Contains(err.Error(), "exactly one of quantity or orderAmount") {
		t.Fatalf("expected exactly-one error, got %v", err)
	}
}

func TestCreateOrderValidatesMarketForbidsPrice(t *testing.T) {
	client := NewClient(nil, "http://localhost", "cid", "secret")
	_, err := client.CreateOrder(context.Background(), "7", OrderCreateRequest{
		Symbol:    "AAPL",
		Side:      "BUY",
		OrderType: "MARKET",
		Quantity:  "1",
		Price:     "100",
	})
	if err == nil || !strings.Contains(err.Error(), "forbids price") {
		t.Fatalf("expected MARKET forbids price error, got %v", err)
	}
}

func TestCreateOrderValidatesLimitRequiresPrice(t *testing.T) {
	client := NewClient(nil, "http://localhost", "cid", "secret")
	_, err := client.CreateOrder(context.Background(), "7", OrderCreateRequest{
		Symbol:    "005930",
		Side:      "BUY",
		OrderType: "LIMIT",
		Quantity:  "1",
	})
	if err == nil || !strings.Contains(err.Error(), "requires price") {
		t.Fatalf("expected LIMIT requires price error, got %v", err)
	}
}

func TestCreateOrderValidatesAmountBasedRequiresMarket(t *testing.T) {
	client := NewClient(nil, "http://localhost", "cid", "secret")
	_, err := client.CreateOrder(context.Background(), "7", OrderCreateRequest{
		Symbol:      "AAPL",
		Side:        "BUY",
		OrderType:   "LIMIT",
		OrderAmount: "100",
		Price:       "10",
	})
	if err == nil || !strings.Contains(err.Error(), "orderType MARKET") {
		t.Fatalf("expected amount-based requires MARKET error, got %v", err)
	}
}

func TestCreateOrderLimitHappyPathIncludesPrice(t *testing.T) {
	var orderBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			writeJSON(t, w, map[string]any{"access_token": "tok", "token_type": "Bearer", "expires_in": 3600})
		case "/api/v1/orders":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			orderBody = string(body)
			writeJSON(t, w, map[string]any{"result": map[string]any{"orderId": "ord-1", "clientOrderId": nil}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.CreateOrder(context.Background(), "7", OrderCreateRequest{
		Symbol:    "005930",
		Side:      "BUY",
		OrderType: "LIMIT",
		Quantity:  "10",
		Price:     "70000",
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if got.OrderID != "ord-1" {
		t.Fatalf("orderId = %q, want ord-1", got.OrderID)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(orderBody), &body); err != nil {
		t.Fatalf("decode order body: %v", err)
	}
	if body["price"] != "70000" {
		t.Fatalf("body missing price: %v", body)
	}
	if body["quantity"] != "10" {
		t.Fatalf("body missing quantity: %v", body)
	}
}

func TestCreateOrderAmountBasedHappyPathOmitsQuantity(t *testing.T) {
	var orderBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			writeJSON(t, w, map[string]any{"access_token": "tok", "token_type": "Bearer", "expires_in": 3600})
		case "/api/v1/orders":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			orderBody = string(body)
			writeJSON(t, w, map[string]any{"result": map[string]any{"orderId": "ord-2", "clientOrderId": nil}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.CreateOrder(context.Background(), "7", OrderCreateRequest{
		Symbol:      "AAPL",
		Side:        "BUY",
		OrderType:   "MARKET",
		OrderAmount: "100.5",
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if got.OrderID != "ord-2" {
		t.Fatalf("orderId = %q, want ord-2", got.OrderID)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(orderBody), &body); err != nil {
		t.Fatalf("decode order body: %v", err)
	}
	if body["orderAmount"] != "100.5" {
		t.Fatalf("body missing orderAmount: %v", body)
	}
	if _, ok := body["quantity"]; ok {
		t.Fatalf("body should omit quantity: %v", body)
	}
}

func TestModifyOrderHappyPath(t *testing.T) {
	var gotPath, gotAccountHeader, gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			writeJSON(t, w, map[string]any{"access_token": "tok", "token_type": "Bearer", "expires_in": 3600})
		default:
			gotPath = r.URL.EscapedPath()
			gotMethod = r.Method
			gotAccountHeader = r.Header.Get("X-Tossinvest-Account")
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			gotBody = string(body)
			writeJSON(t, w, map[string]any{"result": map[string]any{"orderId": "new-op-id"}})
		}
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.ModifyOrder(context.Background(), "7", "order with spaces", OrderModifyRequest{
		OrderType: "LIMIT",
		Quantity:  "15",
		Price:     "71000",
	})
	if err != nil {
		t.Fatalf("ModifyOrder: %v", err)
	}
	if got.OrderID != "new-op-id" {
		t.Fatalf("orderId = %q, want new-op-id", got.OrderID)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	wantPath := "/api/v1/orders/order%20with%20spaces/modify"
	if gotPath != wantPath {
		t.Fatalf("path = %q, want %q", gotPath, wantPath)
	}
	if gotAccountHeader != "7" {
		t.Fatalf("account header = %q, want 7", gotAccountHeader)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(gotBody), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["orderType"] != "LIMIT" || body["quantity"] != "15" || body["price"] != "71000" {
		t.Fatalf("body = %v", body)
	}
}

func TestModifyOrderHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"error":{"code":"modify-restricted","message":"해당 주문은 정정할 수 없습니다.","requestId":"req-1"}}`)
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.ModifyOrder(context.Background(), "7", "ord-1", OrderModifyRequest{OrderType: "LIMIT", Price: "100"})
	if err == nil || !strings.Contains(err.Error(), "modify-restricted") {
		t.Fatalf("expected modify-restricted error, got %v", err)
	}
}

func TestCancelOrderHappyPath(t *testing.T) {
	var gotPath, gotAccountHeader, gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			writeJSON(t, w, map[string]any{"access_token": "tok", "token_type": "Bearer", "expires_in": 3600})
		default:
			gotPath = r.URL.Path
			gotMethod = r.Method
			gotAccountHeader = r.Header.Get("X-Tossinvest-Account")
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			gotBody = string(body)
			writeJSON(t, w, map[string]any{"result": map[string]any{"orderId": "cancel-op-id"}})
		}
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.CancelOrder(context.Background(), "7", "ord-1")
	if err != nil {
		t.Fatalf("CancelOrder: %v", err)
	}
	if got.OrderID != "cancel-op-id" {
		t.Fatalf("orderId = %q, want cancel-op-id", got.OrderID)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/api/v1/orders/ord-1/cancel" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAccountHeader != "7" {
		t.Fatalf("account header = %q, want 7", gotAccountHeader)
	}
	// CancelOrder sends an explicit empty JSON object body, matching the
	// spec's documented `{}` example for the (optional) requestBody, rather
	// than omitting the body entirely.
	if gotBody != "{}" {
		t.Fatalf("body = %q, want {}", gotBody)
	}
}

func TestCancelOrderConflictError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `{"error":{"code":"already-filled","message":"이미 체결된 주문입니다.","requestId":"req-1"}}`)
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.CancelOrder(context.Background(), "7", "ord-1")
	if err == nil || !strings.Contains(err.Error(), "already-filled") {
		t.Fatalf("expected already-filled error, got %v", err)
	}
}

func TestCancelOrderUnprocessableError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"error":{"code":"modify-restricted","message":"해당 주문은 취소할 수 없습니다.","requestId":"req-2"}}`)
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.CancelOrder(context.Background(), "7", "ord-1")
	if err == nil || !strings.Contains(err.Error(), "modify-restricted") {
		t.Fatalf("expected modify-restricted error, got %v", err)
	}
}
