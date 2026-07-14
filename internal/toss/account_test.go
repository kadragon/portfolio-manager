package toss

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestGetAccountsHappyPath(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.RequestURI())
		switch r.URL.Path {
		case "/oauth2/token":
			writeJSON(t, w, map[string]any{
				"access_token": "tok",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case "/api/v1/accounts":
			if got := r.Header.Get("Authorization"); got != "Bearer tok" {
				t.Fatalf("authorization = %q", got)
			}
			writeJSON(t, w, map[string]any{"result": []map[string]any{
				{"accountNo": "12345678901", "accountSeq": 1, "accountType": "BROKERAGE"},
				{"accountNo": "98765432109", "accountSeq": 2, "accountType": "PENSION_SAVINGS"},
			}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetAccounts(context.Background())
	if err != nil {
		t.Fatalf("GetAccounts: %v", err)
	}

	want := []Account{
		{AccountNo: "12345678901", AccountSeq: 1, AccountType: "BROKERAGE"},
		{AccountNo: "98765432109", AccountSeq: 2, AccountType: "PENSION_SAVINGS"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("accounts = %+v, want %+v", got, want)
	}

	wantCalls := []string{
		"POST /oauth2/token",
		"GET /api/v1/accounts",
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", calls, wantCalls)
	}
}

func TestGetAccountsHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/accounts" {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"error":{"code":"FORBIDDEN","message":"no access","requestId":"req-1"}}`)
		}
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := client.GetAccounts(context.Background())
	if err == nil || !strings.Contains(err.Error(), "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN error, got %v", err)
	}
}

func TestGetAccountsEmpty(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/accounts" {
			writeJSON(t, w, map[string]any{"result": []any{}})
		}
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetAccounts(context.Background())
	if err != nil {
		t.Fatalf("GetAccounts: %v", err)
	}
	if got == nil {
		t.Fatal("accounts is nil, want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("accounts = %+v, want empty", got)
	}
}
