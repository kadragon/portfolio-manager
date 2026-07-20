package kis

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDomesticBuyableClient_FetchBuyable(t *testing.T) {
	var gotQuery url.Values
	var gotTrID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		gotTrID = r.Header.Get("tr_id")
		_, _ = w.Write([]byte(`{
			"rt_cd":"0",
			"output":{
				"ord_psbl_cash":"1500000",
				"ord_psbl_sbst":"0",
				"ruse_psbl_amt":"0",
				"nrcvb_buy_amt":"1499000",
				"nrcvb_buy_qty":"20",
				"max_buy_amt":"1499000",
				"max_buy_qty":"20"
			}
		}`))
	}))
	t.Cleanup(srv.Close)

	client := &DomesticBuyableClient{
		HTTP:      srv.Client(),
		BaseURL:   srv.URL,
		AppKey:    "k",
		AppSecret: "s",
		CustType:  "P",
		Env:       "real",
		Manager:   makeManager(t, "token"),
	}

	got, err := client.FetchBuyable("12345678", "01", "005930", "70000", "00")
	if err != nil {
		t.Fatalf("FetchBuyable: %v", err)
	}

	if gotTrID != "TTTC8908R" {
		t.Errorf("tr_id = %q, want TTTC8908R", gotTrID)
	}
	if got := gotQuery.Get("PDNO"); got != "005930" {
		t.Errorf("PDNO = %q, want 005930", got)
	}
	if got := gotQuery.Get("ORD_DVSN"); got != "00" {
		t.Errorf("ORD_DVSN = %q, want 00", got)
	}
	if got := gotQuery.Get("ORD_UNPR"); got != "70000" {
		t.Errorf("ORD_UNPR = %q, want 70000", got)
	}
	if got.OrderableCash.String() != "1500000" {
		t.Errorf("OrderableCash = %s, want 1500000", got.OrderableCash)
	}
	if got.MaxBuyQty.String() != "20" {
		t.Errorf("MaxBuyQty = %s, want 20", got.MaxBuyQty)
	}
}

func TestDomesticBuyableClient_FetchBuyable_DefaultsAndDemoTrID(t *testing.T) {
	var gotQuery url.Values
	var gotTrID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		gotTrID = r.Header.Get("tr_id")
		_, _ = w.Write([]byte(`{"rt_cd":"0","output":{"ord_psbl_cash":"1000"}}`))
	}))
	t.Cleanup(srv.Close)

	client := &DomesticBuyableClient{
		HTTP:      srv.Client(),
		BaseURL:   srv.URL,
		AppKey:    "k",
		AppSecret: "s",
		CustType:  "P",
		Env:       "demo",
		Manager:   makeManager(t, "token"),
	}

	// Empty pdno/ordUnpr/ordDvsn -> cash-only query with defaults.
	got, err := client.FetchBuyable("12345678", "01", "", "", "")
	if err != nil {
		t.Fatalf("FetchBuyable: %v", err)
	}
	if gotTrID != "VTTC8908R" {
		t.Errorf("tr_id = %q, want VTTC8908R (demo)", gotTrID)
	}
	if got := gotQuery.Get("ORD_DVSN"); got != "01" {
		t.Errorf("ORD_DVSN default = %q, want 01", got)
	}
	if got := gotQuery.Get("ORD_UNPR"); got != "0" {
		t.Errorf("ORD_UNPR default = %q, want 0", got)
	}
	if got.OrderableCash.String() != "1000" {
		t.Errorf("OrderableCash = %s, want 1000", got.OrderableCash)
	}
}

func TestDomesticBuyableClient_FetchBuyable_BizError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"rt_cd":"1","msg_cd":"40580000","msg1":"모의투자 조회가 불가합니다"}`))
	}))
	t.Cleanup(srv.Close)

	client := &DomesticBuyableClient{
		HTTP:      srv.Client(),
		BaseURL:   srv.URL,
		AppKey:    "k",
		AppSecret: "s",
		CustType:  "P",
		Env:       "real",
		Manager:   makeManager(t, "token"),
	}

	if _, err := client.FetchBuyable("12345678", "01", "", "", ""); err == nil {
		t.Fatal("FetchBuyable: expected business error, got nil")
	}
}
