package kis

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOverseasOrderClient_PlaceOrder_SendsAccountNumber(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		fmt.Fprint(w, `{"rt_cd":"0"}`)
	}))
	t.Cleanup(srv.Close)

	client := &OverseasOrderClient{
		HTTP:       srv.Client(),
		BaseURL:    srv.URL,
		AppKey:     "k",
		AppSecret:  "s",
		CANO:       "12345678",
		AcntPrdtCd: "01",
		CustType:   "P",
		Env:        "demo",
		Manager:    makeManager(t, "token"),
	}

	if _, err := client.PlaceOrder(context.Background(), "AAPL", "buy", 1, "NASD", ""); err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	if captured["CANO"] != "12345678" {
		t.Errorf("request CANO = %v, want 12345678", captured["CANO"])
	}
	if captured["ACNT_PRDT_CD"] != "01" {
		t.Errorf("request ACNT_PRDT_CD = %v, want 01", captured["ACNT_PRDT_CD"])
	}
}
