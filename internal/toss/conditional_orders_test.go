package toss

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCreateConditionalOrderValidation(t *testing.T) {
	sell := func(trigger string) ConditionRequest {
		return ConditionRequest{OrderSide: "SELL", TriggerPrice: trigger, OrderPrice: trigger}
	}
	buy := func(trigger string) ConditionRequest {
		return ConditionRequest{OrderSide: "BUY", TriggerPrice: trigger, OrderPrice: trigger}
	}

	tests := []struct {
		name    string
		req     ConditionalOrderCreateRequest
		wantErr string
	}{
		{
			name: "SINGLE with second set",
			req: ConditionalOrderCreateRequest{
				Symbol: "005930", Type: "SINGLE", Quantity: "1", OrderType: "LIMIT", ExpireDate: "2026-09-10",
				First:  sell("295"),
				Second: ptr(sell("290")),
			},
			wantErr: "SINGLE must not set second",
		},
		{
			name: "OCO without second",
			req: ConditionalOrderCreateRequest{
				Symbol: "005930", Type: "OCO", Quantity: "1", OrderType: "LIMIT", ExpireDate: "2026-09-10",
				First: sell("305"),
			},
			wantErr: "OCO requires second",
		},
		{
			name: "OCO wrong side",
			req: ConditionalOrderCreateRequest{
				Symbol: "005930", Type: "OCO", Quantity: "1", OrderType: "LIMIT", ExpireDate: "2026-09-10",
				First:  buy("305"),
				Second: ptr(sell("295")),
			},
			wantErr: "OCO requires both legs SELL",
		},
		{
			name: "OCO wrong order type",
			req: ConditionalOrderCreateRequest{
				Symbol: "005930", Type: "OCO", Quantity: "1", OrderType: "MARKET", ExpireDate: "2026-09-10",
				First:  sell("305"),
				Second: ptr(sell("295")),
			},
			wantErr: "OCO requires orderType LIMIT",
		},
		{
			name: "OTO without second",
			req: ConditionalOrderCreateRequest{
				Symbol: "005930", Type: "OTO", Quantity: "1", OrderType: "LIMIT", ExpireDate: "2026-09-10",
				First: buy("290"),
			},
			wantErr: "OTO requires second",
		},
		{
			name: "OTO first not BUY",
			req: ConditionalOrderCreateRequest{
				Symbol: "005930", Type: "OTO", Quantity: "1", OrderType: "LIMIT", ExpireDate: "2026-09-10",
				First:  sell("290"),
				Second: ptr(sell("320")),
			},
			wantErr: "OTO requires first leg BUY",
		},
		{
			name: "OTO second not SELL",
			req: ConditionalOrderCreateRequest{
				Symbol: "005930", Type: "OTO", Quantity: "1", OrderType: "LIMIT", ExpireDate: "2026-09-10",
				First:  buy("290"),
				Second: ptr(buy("320")),
			},
			wantErr: "OTO requires second leg SELL",
		},
		{
			name: "OTO wrong order type",
			req: ConditionalOrderCreateRequest{
				Symbol: "005930", Type: "OTO", Quantity: "1", OrderType: "MARKET", ExpireDate: "2026-09-10",
				First:  buy("290"),
				Second: ptr(sell("320")),
			},
			wantErr: "OTO requires orderType LIMIT",
		},
		{
			name: "unsupported type",
			req: ConditionalOrderCreateRequest{
				Symbol: "005930", Type: "BOGUS", Quantity: "1", OrderType: "LIMIT", ExpireDate: "2026-09-10",
				First: sell("295"),
			},
			wantErr: "unsupported type",
		},
	}

	client := NewClient(nil, "http://localhost", "cid", "secret")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.CreateConditionalOrder(context.Background(), "7", tc.req)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestCreateConditionalOrderSingleHappyPath(t *testing.T) {
	var gotMethod, gotAccount, gotBody string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/conditional-orders" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotMethod = r.Method
		gotAccount = r.Header.Get("X-Tossinvest-Account")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		gotBody = string(body)
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"conditionalOrderId": "cond-1",
			"clientOrderId":      "my-order-001",
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	req := ConditionalOrderCreateRequest{
		Symbol:        "005930",
		Type:          "SINGLE",
		Quantity:      "100",
		OrderType:     "LIMIT",
		ClientOrderID: "my-order-001",
		ExpireDate:    "2026-09-10",
		First:         ConditionRequest{OrderSide: "SELL", TriggerPrice: "295", OrderPrice: "295"},
	}
	got, err := client.CreateConditionalOrder(context.Background(), "7", req)
	if err != nil {
		t.Fatalf("CreateConditionalOrder: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotAccount != "7" {
		t.Fatalf("account header = %q, want 7", gotAccount)
	}
	if got.ConditionalOrderID != "cond-1" || got.ClientOrderID == nil || *got.ClientOrderID != "my-order-001" {
		t.Fatalf("got = %+v", got)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(gotBody), &decoded); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if decoded["type"] != "SINGLE" || decoded["symbol"] != "005930" {
		t.Fatalf("body = %v", decoded)
	}
	if _, ok := decoded["second"]; ok {
		t.Fatalf("body should omit second, got %v", decoded)
	}
}

func TestCreateConditionalOrderOCOHappyPath(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/conditional-orders" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"conditionalOrderId": "cond-oco-1",
			"clientOrderId":      nil,
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.CreateConditionalOrder(context.Background(), "7", ConditionalOrderCreateRequest{
		Symbol:     "005930",
		Type:       "OCO",
		Quantity:   "100",
		OrderType:  "LIMIT",
		ExpireDate: "2026-09-10",
		First:      ConditionRequest{OrderSide: "SELL", TriggerPrice: "305", OrderPrice: "305"},
		Second:     ptr(ConditionRequest{OrderSide: "SELL", TriggerPrice: "295", OrderPrice: "294.5"}),
	})
	if err != nil {
		t.Fatalf("CreateConditionalOrder: %v", err)
	}
	if got.ConditionalOrderID != "cond-oco-1" || got.ClientOrderID != nil {
		t.Fatalf("got = %+v", got)
	}
}

func TestModifyConditionalOrderHappyPath(t *testing.T) {
	var gotMethod, gotPath, gotAccount string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAccount = r.Header.Get("X-Tossinvest-Account")
		writeJSON(t, w, map[string]any{"result": map[string]any{"conditionalOrderId": "cond-2"}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.ModifyConditionalOrder(context.Background(), "7", "cond-1/x", ConditionalOrderModifyRequest{
		Type:       "OCO",
		Quantity:   "100",
		OrderType:  "LIMIT",
		ExpireDate: "2026-09-10",
		First:      ConditionRequest{OrderSide: "SELL", TriggerPrice: "310", OrderPrice: "310"},
		Second:     ptr(ConditionRequest{OrderSide: "SELL", TriggerPrice: "290", OrderPrice: "290"}),
	})
	if err != nil {
		t.Fatalf("ModifyConditionalOrder: %v", err)
	}
	if got.ConditionalOrderID != "cond-2" {
		t.Fatalf("got = %+v", got)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	wantPath := "/api/v1/conditional-orders/cond-1%2Fx/modify"
	if gotPath != wantPath {
		t.Fatalf("path = %s, want %s", gotPath, wantPath)
	}
	if gotAccount != "7" {
		t.Fatalf("account header = %q, want 7", gotAccount)
	}
}

func TestModifyConditionalOrderHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"code":"conditional-order-not-found","message":"조건주문을 찾을 수 없습니다."}}`)
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.ModifyConditionalOrder(context.Background(), "7", "cond-1", ConditionalOrderModifyRequest{
		Type:       "SINGLE",
		Quantity:   "1",
		OrderType:  "LIMIT",
		ExpireDate: "2026-09-10",
		First:      ConditionRequest{OrderSide: "SELL", TriggerPrice: "295", OrderPrice: "295"},
	})
	if err == nil || !strings.Contains(err.Error(), "conditional-order-not-found") {
		t.Fatalf("expected conditional-order-not-found error, got %v", err)
	}
}

func TestCancelConditionalOrderHappyPath(t *testing.T) {
	var gotMethod, gotPath, gotAccount string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAccount = r.Header.Get("X-Tossinvest-Account")
		w.WriteHeader(http.StatusNoContent)
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	if err := client.CancelConditionalOrder(context.Background(), "7", "cond-1"); err != nil {
		t.Fatalf("CancelConditionalOrder: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method = %s, want DELETE", gotMethod)
	}
	if gotPath != "/api/v1/conditional-orders/cond-1" {
		t.Fatalf("path = %s", gotPath)
	}
	if gotAccount != "7" {
		t.Fatalf("account header = %q, want 7", gotAccount)
	}
}

func TestCancelConditionalOrderHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"code":"conditional-order-not-found","message":"조건주문을 찾을 수 없습니다."}}`)
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	err := client.CancelConditionalOrder(context.Background(), "7", "cond-1")
	if err == nil || !strings.Contains(err.Error(), "conditional-order-not-found") {
		t.Fatalf("expected conditional-order-not-found error, got %v", err)
	}
}

func ptr[T any](v T) *T { return &v }
