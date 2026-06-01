package kis

// TestLiveOverseasPriceRaw sends a real request to the KIS price endpoint
// and dumps the raw response so the root cause can be confirmed.
//
// Gate: only runs when KIS_LIVE=1. Never in CI.
// Token reuses the cached .data/kis_token_1.json (no forced refresh).
//
// Run:
//
//	set -a; . ./.env; set +a
//	KIS_LIVE=1 go test ./internal/kis/ -run TestLiveOverseasPriceRaw -v

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestLiveOverseasPriceRaw(t *testing.T) {
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

	// Reuse cached token; only refreshes if expired. No forced refresh.
	// Path relative to repo root (go test runs in package dir, so two levels up).
	store := NewFileTokenStore("../../.data/kis_token_1.json")
	authBaseURL := "https://openapi.koreainvestment.com:9443"
	if env == "demo" || env == "vps" || env == "paper" {
		authBaseURL = "https://openapivts.koreainvestment.com:29443"
	}
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
	pc := &OverseasPriceClient{
		HTTP:      http.DefaultClient,
		BaseURL:   baseURL,
		AppKey:    appKey,
		AppSecret: appSecret,
		CustType:  custType,
		Env:       env,
		Manager:   mgr,
	}

	cases := []struct {
		excd   string // canonical form; converted to short form internally
		ticker string
	}{
		{"NASD", "QQQ"},
		{"AMEX", "SPY"},
	}

	for _, tc := range cases {
		t.Run(tc.ticker+"@"+tc.excd, func(t *testing.T) {
			wireEXCD := shortExchangeCode(tc.excd)
			trID, err := TrIDForEnv(env, "HHDFS00000300", "HHDFS00000300")
			if err != nil {
				t.Fatalf("TrIDForEnv: %v", err)
			}
			token, err := mgr.GetToken()
			if err != nil {
				t.Fatalf("GetToken: %v", err)
			}

			// Raw GET — same path as FetchCurrentPrice but returns body before parsing.
			body, status, err := doGet(
				pc.HTTP,
				pc.BaseURL+"/uapi/overseas-price/v1/quotations/price",
				map[string]string{
					"AUTH": "",
					"EXCD": wireEXCD,
					"SYMB": tc.ticker,
				},
				BuildHeaders(token, appKey, appSecret, trID, custType),
			)
			if err != nil {
				t.Fatalf("HTTP error: %v", err)
			}

			// Pretty-print top-level fields for triage.
			var top map[string]json.RawMessage
			if json.Unmarshal(body, &top) != nil {
				t.Logf("raw body (non-JSON): %s", body)
				return
			}

			rtCd := unquote(top["rt_cd"])
			msgCd := unquote(top["msg_cd"])
			msg1 := unquote(top["msg1"])

			fmt.Printf("\n=== %s@%s (wire EXCD=%s, HTTP %d) ===\n", tc.ticker, tc.excd, wireEXCD, status)
			fmt.Printf("  rt_cd : %s\n", rtCd)
			fmt.Printf("  msg_cd: %s\n", msgCd)
			fmt.Printf("  msg1  : %s\n", msg1)

			// Decode output for detailed field dump.
			if outputRaw, ok := top["output"]; ok {
				var out map[string]string
				if json.Unmarshal(outputRaw, &out) == nil {
					interesting := []string{"last", "base", "ordy", "name", "enname", "symbol", "symb"}
					for _, k := range interesting {
						if v, exists := out[k]; exists {
							fmt.Printf("  output.%-8s = %q\n", k, v)
						}
					}
					// Full output for any unknown field.
					fmt.Printf("  output (full): ")
					if enc, err2 := json.Marshal(out); err2 == nil {
						fmt.Printf("%s\n", enc)
					}
				} else {
					fmt.Printf("  output: %s\n", outputRaw)
				}
			}

			// Assertions for Phase 1 diagnosis.
			if rtCd != "0" {
				t.Errorf("rt_cd=%q (not 0): msg_cd=%q msg1=%q — API error, not a parser issue", rtCd, msgCd, msg1)
				return
			}
			quote := ParseUSPrice(body, tc.ticker, tc.excd)
			fmt.Printf("  parsed price : %v\n", quote.Price)
			fmt.Printf("  parsed name  : %q\n", quote.Name)
			if quote.Price <= 0 {
				// Identify last/base values for the parser-bug diagnosis.
				var top2 map[string]json.RawMessage
				_ = json.Unmarshal(body, &top2)
				var out map[string]string
				if outputRaw, ok2 := top2["output"]; ok2 {
					_ = json.Unmarshal(outputRaw, &out)
				}
				t.Errorf("price=0 after parsing: last=%q base=%q — if base>0, parser needs base fallback", out["last"], out["base"])
			}
		})
	}
}

// unquote strips JSON string quotes; returns raw bytes as string on error.
func unquote(raw json.RawMessage) string {
	if raw == nil {
		return "<absent>"
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return string(raw)
}
