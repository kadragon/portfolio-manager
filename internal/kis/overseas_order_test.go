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

func TestOverseasOrderClient_PlaceOrder_Market(t *testing.T) {
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

	if captured["ORD_DVSN"] != "01" {
		t.Errorf("ORD_DVSN = %v, want 01 (market)", captured["ORD_DVSN"])
	}
	if captured["OVRS_ORD_UNPR"] != "0" {
		t.Errorf("OVRS_ORD_UNPR = %v, want 0", captured["OVRS_ORD_UNPR"])
	}
}

func TestOverseasOrderClient_PlaceOrder_Limit(t *testing.T) {
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

	if _, err := client.PlaceOrder(context.Background(), "AAPL", "buy", 3, "NASD", "195.89"); err != nil {
		t.Fatalf("PlaceOrder limit: %v", err)
	}

	if captured["ORD_DVSN"] != "00" {
		t.Errorf("ORD_DVSN = %v, want 00 (limit)", captured["ORD_DVSN"])
	}
	if captured["OVRS_ORD_UNPR"] != "195.89" {
		t.Errorf("OVRS_ORD_UNPR = %v, want 195.89", captured["OVRS_ORD_UNPR"])
	}
}
