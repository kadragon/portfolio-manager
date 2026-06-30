package kis

// TestLiveOverseasInfoSecurityGroup sends real requests to the KIS overseas
// search-info endpoint and verifies that ClassifyAssetClass + OverseasSecurityGroup
// produce the expected FE/FS codes for a known ETF and a known stock.
//
// Gate: only runs when KIS_LIVE=1. Never in CI.
// Token reuses the cached .data/kis_token_1.json (no forced refresh).
//
// Run:
//
//	set -a; . ./.env; set +a
//	KIS_LIVE=1 go test ./internal/kis/ -run TestLiveOverseasInfoSecurityGroup -v

import (
	"net/http"
	"os"
	"testing"
	"time"
)

func TestLiveOverseasInfoSecurityGroup(t *testing.T) {
	if os.Getenv("KIS_LIVE") != "1" {
		t.Skip("set KIS_LIVE=1 to run live KIS integration tests")
	}

	appKey := os.Getenv("KIS_APP_KEY")
	appSecret := os.Getenv("KIS_APP_SECRET")
	custType := os.Getenv("KIS_CUST_TYPE")
	env := os.Getenv("KIS_ENV")
	if appKey == "" || appSecret == "" {
		t.Fatal("KIS_APP_KEY and KIS_APP_SECRET must be set")
	}
	if custType == "" {
		custType = "P"
	}
	if env == "" {
		env = "real"
	}

	authBaseURL := "https://openapi.koreainvestment.com:9443"
	if env == "demo" || env == "vps" || env == "paper" {
		authBaseURL = "https://openapivts.koreainvestment.com:29443"
	}
	store := NewFileTokenStore("../../.data/kis_token_1.json")
	authClient := &AuthClient{
		HTTPClient: http.DefaultClient,
		BaseURL:    authBaseURL,
		AppKey:     appKey,
		AppSecret:  appSecret,
	}
	mgr := NewTokenManager(store, authClient, time.Minute)

	baseURL := "https://openapi.koreainvestment.com:9443"
	if env == "demo" || env == "vps" || env == "paper" {
		baseURL = "https://openapivts.koreainvestment.com:29443"
	}
	trID := os.Getenv("KIS_OVERSEAS_INFO_TR_ID")
	if trID == "" {
		trID = "CTPF1702R"
	}
	ic := &OverseasInfoClient{
		HTTP:      http.DefaultClient,
		BaseURL:   baseURL,
		AppKey:    appKey,
		AppSecret: appSecret,
		TrID:      trID,
		CustType:  custType,
		Manager:   mgr,
	}

	cases := []struct {
		excd       string
		ticker     string
		wantClass  string
		wantSGCode string
	}{
		{"NAS", "QQQ", "etf", "FE"},
		{"NAS", "AAPL", "stock", "FS"},
	}

	for _, tc := range cases {
		t.Run(tc.ticker+"@"+tc.excd, func(t *testing.T) {
			assetClass, err := ic.ClassifyAssetClass(tc.excd, tc.ticker)
			if err != nil {
				t.Fatalf("ClassifyAssetClass: %v", err)
			}
			if assetClass != tc.wantClass {
				t.Errorf("ClassifyAssetClass = %q, want %q", assetClass, tc.wantClass)
			}
			sg := OverseasSecurityGroup(assetClass)
			if sg != tc.wantSGCode {
				t.Errorf("OverseasSecurityGroup(%q) = %q, want %q", assetClass, sg, tc.wantSGCode)
			}
			t.Logf("%s@%s: asset_class=%q security_group=%q", tc.ticker, tc.excd, assetClass, sg)
		})
	}
}
