package kis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kadragon/portfolio-manager/internal/datex"
)

// ---------- helpers ----------

func makeManager(t *testing.T, token string) *TokenManager {
	t.Helper()
	store := &MemoryTokenStore{}
	_ = store.Save(token, time.Now().Add(24*time.Hour))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"access_token":"refreshed","expires_in":86400}`)
	}))
	t.Cleanup(srv.Close)
	auth := &AuthClient{HTTPClient: srv.Client(), BaseURL: srv.URL, AppKey: "k", AppSecret: "s"}
	return NewTokenManager(store, auth, time.Minute)
}

func makeClient(t *testing.T, handler http.Handler) (*http.Client, string) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv.Client(), srv.URL
}

func TestIsDomesticTicker(t *testing.T) {
	cases := []struct {
		ticker   string
		domestic bool
	}{
		{"005930", true}, // Samsung Electronics
		{"0052D0", true}, // 6-char domestic ETF
		{"AAPL", false},
		{"TSLA", false},
		{"VOO", false},
		{"", false},
		{"1234567", false}, // 7 chars
	}
	for _, tc := range cases {
		if got := IsDomesticTicker(tc.ticker); got != tc.domestic {
			t.Errorf("IsDomesticTicker(%q) = %v, want %v", tc.ticker, got, tc.domestic)
		}
	}
}

func TestIsTokenExpiredError(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   bool
	}{
		{500, `{"msg_cd":"EGW00123","msg1":"token expired"}`, true},
		{500, `{"msg_cd":"OTHER","msg1":"other error"}`, false},
		{200, `{"msg_cd":"EGW00123"}`, false}, // must be 500
		{500, `not json`, false},
	}
	for _, tc := range cases {
		got := IsTokenExpiredError(tc.status, []byte(tc.body))
		if got != tc.want {
			t.Errorf("IsTokenExpiredError(%d, %q) = %v, want %v", tc.status, tc.body, got, tc.want)
		}
	}
}

func TestParseKoreaPrice(t *testing.T) {
	data := []byte(`{
		"output": {
			"stck_code": "005930",
			"hts_kor_isnm": "삼성전자",
			"stck_prpr": "74000"
		}
	}`)
	q := ParseKoreaPrice(data, "005930")
	if q.Symbol != "005930" {
		t.Errorf("Symbol = %q, want 005930", q.Symbol)
	}
	if q.Name != "삼성전자" {
		t.Errorf("Name = %q, want 삼성전자", q.Name)
	}
	if q.Price != 74000 {
		t.Errorf("Price = %v, want 74000", q.Price)
	}
	if q.Currency != "KRW" {
		t.Errorf("Currency = %q, want KRW", q.Currency)
	}
}

func TestParseKoreaPriceArrayOutput(t *testing.T) {
	// KIS sometimes returns output as an array
	data := []byte(`{"output":[{"stck_code":"005930","hts_kor_isnm":"삼성전자","stck_prpr":"74000"}]}`)
	q := ParseKoreaPrice(data, "005930")
	if q.Price != 74000 {
		t.Errorf("Price from array output = %v, want 74000", q.Price)
	}
}

func TestParseKoreaPriceFallbackSymbol(t *testing.T) {
	// stck_code absent → use fallback symbol
	data := []byte(`{"output":{"hts_kor_isnm":"삼성전자","stck_prpr":"74000"}}`)
	q := ParseKoreaPrice(data, "FALLBACK")
	if q.Symbol != "FALLBACK" {
		t.Errorf("Symbol fallback = %q, want FALLBACK", q.Symbol)
	}
}

func TestParseUSPrice(t *testing.T) {
	data := []byte(`{
		"output": {
			"symbol": "AAPL",
			"name": "Apple Inc.",
			"last": "195.89"
		}
	}`)
	q := ParseUSPrice(data, "AAPL", "NASD")
	if q.Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want AAPL", q.Symbol)
	}
	if q.Name != "Apple Inc." {
		t.Errorf("Name = %q, want Apple Inc.", q.Name)
	}
	if q.Price != 195.89 {
		t.Errorf("Price = %v, want 195.89", q.Price)
	}
	if q.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", q.Currency)
	}
	if q.Exchange != "NASD" {
		t.Errorf("Exchange = %q, want NASD", q.Exchange)
	}
}

func TestParseUSPriceNameFallback(t *testing.T) {
	// name key absent, falls back through the key list
	data := []byte(`{"output":{"symbol":"TSLA","prdt_name":"Tesla Inc","last":"250.00"}}`)
	q := ParseUSPrice(data, "TSLA", "NASD")
	if q.Name != "Tesla Inc" {
		t.Errorf("Name fallback = %q, want Tesla Inc", q.Name)
	}
}

func TestParseUSPriceSymbolFallbacks(t *testing.T) {
	// symbol → symb → rsym → fallback
	data := []byte(`{"output":{"symb":"VOO","last":"420.00"}}`)
	q := ParseUSPrice(data, "VOO", "NYSE")
	if q.Symbol != "VOO" {
		t.Errorf("Symbol from symb = %q, want VOO", q.Symbol)
	}

	data2 := []byte(`{"output":{"rsym":"VTI","last":"200.00"}}`)
	q2 := ParseUSPrice(data2, "VTI", "NYSE")
	if q2.Symbol != "VTI" {
		t.Errorf("Symbol from rsym = %q, want VTI", q2.Symbol)
	}
}

func TestTrIDForEnv(t *testing.T) {
	cases := []struct {
		env    string
		wantID string
		wantOK bool
	}{
		{"real", "REAL_ID", true},
		{"prod", "REAL_ID", true},
		{"REAL", "REAL_ID", true},
		{"real/extra", "REAL_ID", true},
		{"demo", "DEMO_ID", true},
		{"vps", "DEMO_ID", true},
		{"paper", "DEMO_ID", true},
		{"unknown", "", false},
	}
	for _, tc := range cases {
		got, err := TrIDForEnv(tc.env, "REAL_ID", "DEMO_ID")
		if tc.wantOK && err != nil {
			t.Errorf("TrIDForEnv(%q) unexpected error: %v", tc.env, err)
		}
		if !tc.wantOK && err == nil {
			t.Errorf("TrIDForEnv(%q) expected error, got %q", tc.env, got)
		}
		if tc.wantOK && got != tc.wantID {
			t.Errorf("TrIDForEnv(%q) = %q, want %q", tc.env, got, tc.wantID)
		}
	}
}

func TestPrioritizedExchanges(t *testing.T) {
	cases := []struct {
		preferred string
		wantFirst string
	}{
		{"NASD", "NASD"},
		{"NYSE", "NYSE"},
		{"AMEX", "AMEX"},
		{"", "NASD"},
		{"UNKNOWN", "NASD"}, // fallback to default order
	}
	for _, tc := range cases {
		got := prioritizedExchanges(tc.preferred)
		if len(got) == 0 || got[0] != tc.wantFirst {
			t.Errorf("prioritizedExchanges(%q)[0] = %q, want %q (full=%v)", tc.preferred, got[0], tc.wantFirst, got)
		}
	}
}

// ---------- token.go ----------

func TestMemoryTokenStoreSaveLoad(t *testing.T) {
	store := &MemoryTokenStore{}

	// empty store returns nil
	got, err := store.Load()
	if err != nil || got != nil {
		t.Fatalf("empty Load() = %v, %v; want nil, nil", got, err)
	}

	expiry := time.Now().Add(time.Hour).Truncate(time.Second)
	if err := store.Save("tok123", expiry); err != nil {
		t.Fatalf("Save: %v", err)
	}
	td, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if td.Token != "tok123" {
		t.Errorf("Token = %q, want tok123", td.Token)
	}
	if !td.ExpiresAt.Equal(expiry) {
		t.Errorf("ExpiresAt = %v, want %v", td.ExpiresAt, expiry)
	}
}

func TestFileTokenStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	store := NewFileTokenStore(path)

	// nonexistent → nil
	got, err := store.Load()
	if err != nil || got != nil {
		t.Fatalf("nonexistent Load() = %v, %v; want nil, nil", got, err)
	}

	expiry := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	if err := store.Save("file_tok", expiry); err != nil {
		t.Fatalf("Save: %v", err)
	}

	td, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if td.Token != "file_tok" {
		t.Errorf("Token = %q, want file_tok", td.Token)
	}
	// Round-trip through KST; compare Unix second
	if td.ExpiresAt.Unix() != expiry.Unix() {
		t.Errorf("ExpiresAt unix = %d, want %d", td.ExpiresAt.Unix(), expiry.Unix())
	}
}

func TestFileTokenStoreLoadPythonFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "py_token.json")
	// Python-style naive datetime string
	raw := `{"token":"pytok","expires_at":"2026-01-03 13:21:44"}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewFileTokenStore(path)
	td, err := store.Load()
	if err != nil {
		t.Fatalf("Load python format: %v", err)
	}
	if td.Token != "pytok" {
		t.Errorf("Token = %q, want pytok", td.Token)
	}
}

func TestParseTokenTime(t *testing.T) {
	cases := []struct {
		input   string
		wantErr bool
	}{
		{"2026-01-03T13:21:44+09:00", false}, // RFC3339 with offset
		{"2026-01-03T13:21:44Z", false},      // RFC3339 UTC
		{"2026-01-03T13:21:44", false},       // naive ISO
		{"2026-01-03 13:21:44", false},       // Python naive
		{"not-a-date", true},
	}
	for _, tc := range cases {
		_, err := parseTokenTime(tc.input)
		if tc.wantErr && err == nil {
			t.Errorf("parseTokenTime(%q) expected error", tc.input)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("parseTokenTime(%q) unexpected error: %v", tc.input, err)
		}
	}
}

// ---------- auth.go ----------

func TestAuthClientRequestAccessTokenHappy(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth2/tokenP" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		fmt.Fprintf(w, `{"access_token":"tok123","expires_in":86400}`)
	}))
	auth := &AuthClient{HTTPClient: client, BaseURL: baseURL, AppKey: "k", AppSecret: "s"}
	td, err := auth.RequestAccessToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if td.Token != "tok123" {
		t.Errorf("Token = %q, want tok123", td.Token)
	}
	if td.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should not be zero")
	}
}

func TestAuthClientRequestAccessTokenHTTP400(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"bad request"}`)
	}))
	auth := &AuthClient{HTTPClient: client, BaseURL: baseURL, AppKey: "k", AppSecret: "s"}
	_, err := auth.RequestAccessToken()
	if err == nil {
		t.Fatal("expected error for HTTP 400")
	}
}

func TestAuthClientRequestAccessTokenExpiredField(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"access_token":"tok_exp","access_token_token_expired":"2026-01-03 13:21:44"}`)
	}))
	auth := &AuthClient{HTTPClient: client, BaseURL: baseURL, AppKey: "k", AppSecret: "s"}
	td, err := auth.RequestAccessToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if td.Token != "tok_exp" {
		t.Errorf("Token = %q, want tok_exp", td.Token)
	}
	if td.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should not be zero when expires field provided")
	}
}

func TestAuthClientRequestAccessTokenInvalidJSON(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `not json`)
	}))
	auth := &AuthClient{HTTPClient: client, BaseURL: baseURL, AppKey: "k", AppSecret: "s"}
	_, err := auth.RequestAccessToken()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ---------- token_manager.go ----------

func TestTokenManagerGetTokenCacheEmpty(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		fmt.Fprintf(w, `{"access_token":"fresh_tok","expires_in":86400}`)
	}))
	t.Cleanup(srv.Close)

	store := &MemoryTokenStore{}
	auth := &AuthClient{HTTPClient: srv.Client(), BaseURL: srv.URL, AppKey: "k", AppSecret: "s"}
	mgr := NewTokenManager(store, auth, time.Minute)

	tok, err := mgr.GetToken()
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if tok != "fresh_tok" {
		t.Errorf("token = %q, want fresh_tok", tok)
	}
	if callCount != 1 {
		t.Errorf("auth calls = %d, want 1", callCount)
	}
}

func TestTokenManagerGetTokenCacheHit(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		fmt.Fprintf(w, `{"access_token":"refreshed","expires_in":86400}`)
	}))
	t.Cleanup(srv.Close)

	store := &MemoryTokenStore{}
	_ = store.Save("cached_tok", time.Now().Add(24*time.Hour))
	auth := &AuthClient{HTTPClient: srv.Client(), BaseURL: srv.URL, AppKey: "k", AppSecret: "s"}
	mgr := NewTokenManager(store, auth, time.Minute)

	tok, err := mgr.GetToken()
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if tok != "cached_tok" {
		t.Errorf("token = %q, want cached_tok", tok)
	}
	if callCount != 0 {
		t.Errorf("auth calls = %d, want 0 (should use cache)", callCount)
	}
}

func TestTokenManagerGetTokenWithinSkew(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		fmt.Fprintf(w, `{"access_token":"new_tok","expires_in":86400}`)
	}))
	t.Cleanup(srv.Close)

	store := &MemoryTokenStore{}
	// Token expires in 30 seconds, skew is 1 minute → should refresh
	_ = store.Save("expiring_tok", time.Now().Add(30*time.Second))
	auth := &AuthClient{HTTPClient: srv.Client(), BaseURL: srv.URL, AppKey: "k", AppSecret: "s"}
	mgr := NewTokenManager(store, auth, time.Minute)

	tok, err := mgr.GetToken()
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if tok != "new_tok" {
		t.Errorf("token = %q, want new_tok", tok)
	}
	if callCount != 1 {
		t.Errorf("auth calls = %d, want 1 (should refresh within skew)", callCount)
	}
}

func TestTokenManagerRefreshToken(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		fmt.Fprintf(w, `{"access_token":"forced_refresh","expires_in":86400}`)
	}))
	t.Cleanup(srv.Close)

	store := &MemoryTokenStore{}
	_ = store.Save("valid_tok", time.Now().Add(24*time.Hour))
	auth := &AuthClient{HTTPClient: srv.Client(), BaseURL: srv.URL, AppKey: "k", AppSecret: "s"}
	mgr := NewTokenManager(store, auth, time.Minute)

	tok, err := mgr.RefreshToken()
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if tok != "forced_refresh" {
		t.Errorf("token = %q, want forced_refresh", tok)
	}
	if callCount != 1 {
		t.Errorf("auth calls = %d, want 1", callCount)
	}
}

// ---------- base_client.go ----------

func TestGetWithRetryHappy(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"result":"ok"}`)
	}))
	mgr := makeManager(t, "test_token")
	body, err := GetWithRetry(client, baseURL+"/test", nil,
		BuildHeaders("test_token", "k", "s", "TRID", "P"),
		mgr, "k", "s", "TRID", "P")
	if err != nil {
		t.Fatalf("GetWithRetry: %v", err)
	}
	if !strings.Contains(string(body), "ok") {
		t.Errorf("body = %q, want to contain 'ok'", body)
	}
}

func TestGetWithRetryTokenExpiry(t *testing.T) {
	callCount := 0
	authCallCount := 0
	// Single server handles both the resource endpoint and the auth refresh
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/tokenP" {
			authCallCount++
			fmt.Fprintf(w, `{"access_token":"refreshed","expires_in":86400}`)
			return
		}
		callCount++
		if callCount == 1 {
			w.WriteHeader(500)
			fmt.Fprintf(w, `{"msg_cd":"EGW00123","msg1":"token expired"}`)
			return
		}
		fmt.Fprintf(w, `{"result":"after_retry"}`)
	}))
	t.Cleanup(srv.Close)

	store := &MemoryTokenStore{}
	_ = store.Save("old_token", time.Now().Add(24*time.Hour))
	auth := &AuthClient{HTTPClient: srv.Client(), BaseURL: srv.URL, AppKey: "k", AppSecret: "s"}
	mgr := NewTokenManager(store, auth, time.Minute)

	body, err := GetWithRetry(srv.Client(), srv.URL+"/resource", nil,
		BuildHeaders("old_token", "k", "s", "TRID", "P"),
		mgr, "k", "s", "TRID", "P")
	if err != nil {
		t.Fatalf("GetWithRetry: %v", err)
	}
	if !strings.Contains(string(body), "after_retry") {
		t.Errorf("body = %q, want 'after_retry'", body)
	}
	if authCallCount < 1 {
		t.Errorf("auth called %d times, want ≥1", authCallCount)
	}
}

func TestGetWithRetryHTTP400(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"bad"}`)
	}))
	mgr := makeManager(t, "tok")
	_, err := GetWithRetry(client, baseURL+"/test", nil,
		BuildHeaders("tok", "k", "s", "TRID", "P"),
		mgr, "k", "s", "TRID", "P")
	if err == nil {
		t.Fatal("expected error for HTTP 400")
	}
}

func TestGetWithRetryFull(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "header_val")
		fmt.Fprintf(w, `{"result":"full"}`)
	}))
	mgr := makeManager(t, "tok")
	body, hdrs, err := GetWithRetryFull(client, baseURL+"/test", nil,
		BuildHeaders("tok", "k", "s", "TRID", "P"),
		mgr, "k", "s", "TRID", "P")
	if err != nil {
		t.Fatalf("GetWithRetryFull: %v", err)
	}
	if !strings.Contains(string(body), "full") {
		t.Errorf("body = %q, want 'full'", body)
	}
	if hdrs.Get("X-Custom") != "header_val" {
		t.Errorf("X-Custom header = %q, want header_val", hdrs.Get("X-Custom"))
	}
}

func TestPostWithRetryHappy(t *testing.T) {
	var received map[string]any
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode body: %v", err)
		}
		fmt.Fprintf(w, `{"rt_cd":"0"}`)
	}))
	mgr := makeManager(t, "tok")
	payload := map[string]string{"PDNO": "005930"}
	body, err := postWithRetry(client, baseURL+"/order", payload,
		BuildHeaders("tok", "k", "s", "TRID", "P"),
		mgr, "k", "s", "TRID", "P")
	if err != nil {
		t.Fatalf("postWithRetry: %v", err)
	}
	if !strings.Contains(string(body), "rt_cd") {
		t.Errorf("body = %q, want 'rt_cd'", body)
	}
	if received["PDNO"] != "005930" {
		t.Errorf("received PDNO = %v, want 005930", received["PDNO"])
	}
}

func TestBuildHeaders(t *testing.T) {
	hdrs := BuildHeaders("mytok", "mykey", "mysecret", "MYTRID", "P")
	cases := map[string]string{
		"authorization": "Bearer mytok",
		"appkey":        "mykey",
		"appsecret":     "mysecret",
		"tr_id":         "MYTRID",
		"custtype":      "P",
		"content-type":  "application/json",
	}
	for k, want := range cases {
		if got := hdrs[k]; got != want {
			t.Errorf("headers[%q] = %q, want %q", k, got, want)
		}
	}
}

// ---------- domestic_price.go ----------

func TestDomesticPriceClientFetchCurrentPrice(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"output":{"stck_code":"005930","hts_kor_isnm":"삼성전자","stck_prpr":"74000"}}`)
	}))
	mgr := makeManager(t, "test_tok")
	pc := &DomesticPriceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	q, err := pc.FetchCurrentPrice("J", "005930")
	if err != nil {
		t.Fatalf("FetchCurrentPrice: %v", err)
	}
	if q.Symbol != "005930" {
		t.Errorf("Symbol = %q, want 005930", q.Symbol)
	}
	if q.Price != 74000 {
		t.Errorf("Price = %v, want 74000", q.Price)
	}
}

func TestDomesticPriceClientFetchHistoricalClose(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"output2":[{"stck_bsop_date":"20240101","stck_clpr":"73000"}]}`)
	}))
	mgr := makeManager(t, "test_tok")
	pc := &DomesticPriceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	price, err := pc.FetchHistoricalClose("005930", datex.New(2024, 1, 1))
	if err != nil {
		t.Fatalf("FetchHistoricalClose: %v", err)
	}
	if price != 73000.0 {
		t.Errorf("price = %v, want 73000.0", price)
	}
}

func TestParseDomesticHistoricalExactMatch(t *testing.T) {
	data := []byte(`{"output2":[{"stck_bsop_date":"20240101","stck_clpr":"73000"},{"stck_bsop_date":"20231231","stck_clpr":"72000"}]}`)
	price := parseDomesticHistorical(data, "20240101")
	if price != 73000.0 {
		t.Errorf("price = %v, want 73000.0", price)
	}
}

func TestParseDomesticHistoricalFallback(t *testing.T) {
	// No exact match — returns first item
	data := []byte(`{"output2":[{"stck_bsop_date":"20231231","stck_clpr":"72000"}]}`)
	price := parseDomesticHistorical(data, "20240101")
	if price != 72000.0 {
		t.Errorf("price = %v, want 72000.0", price)
	}
}

func TestParseDomesticHistoricalEmpty(t *testing.T) {
	data := []byte(`{"output2":[]}`)
	price := parseDomesticHistorical(data, "20240101")
	if price != 0 {
		t.Errorf("price = %v, want 0", price)
	}
}

// ---------- domestic_balance.go ----------

func TestRaiseBizError(t *testing.T) {
	errData := map[string]any{"rt_cd": "1", "msg_cd": "ERR01", "msg1": "error message"}
	if err := raiseBizError(errData); err == nil {
		t.Error("expected error for rt_cd=1")
	}

	okData := map[string]any{"rt_cd": "0", "msg_cd": "OK", "msg1": "success"}
	if err := raiseBizError(okData); err != nil {
		t.Errorf("expected nil for rt_cd=0, got: %v", err)
	}

	emptyData := map[string]any{}
	if err := raiseBizError(emptyData); err != nil {
		t.Errorf("expected nil for empty data, got: %v", err)
	}
}

func TestParseCashBalance(t *testing.T) {
	// First key present → returns its value even if "0"
	row := map[string]any{"dnca_tot_amt": "500000", "ord_psbl_cash": "300000"}
	got := parseCashBalance(row)
	if got.String() != "500000" {
		t.Errorf("dnca_tot_amt key: got %v, want 500000", got)
	}

	// dnca_tot_amt = "0" → returns 0 (key-existence wins)
	row2 := map[string]any{"dnca_tot_amt": "0", "ord_psbl_cash": "300000"}
	got2 := parseCashBalance(row2)
	if got2.String() != "0" {
		t.Errorf("dnca_tot_amt=0 key: got %v, want 0", got2)
	}

	// ord_psbl_cash fallback
	row3 := map[string]any{"ord_psbl_cash": "200000"}
	got3 := parseCashBalance(row3)
	if got3.String() != "200000" {
		t.Errorf("ord_psbl_cash fallback: got %v, want 200000", got3)
	}

	// tot_dnca_amt fallback
	row4 := map[string]any{"tot_dnca_amt": "100000"}
	got4 := parseCashBalance(row4)
	if got4.String() != "100000" {
		t.Errorf("tot_dnca_amt fallback: got %v, want 100000", got4)
	}

	// No key → zero
	row5 := map[string]any{"other_key": "999"}
	got5 := parseCashBalance(row5)
	if got5.String() != "0" {
		t.Errorf("no key: got %v, want 0", got5)
	}
}

func TestFetchAccountSnapshotSinglePage(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// tr_cont header absent → single page
		fmt.Fprintf(w, `{
			"rt_cd":"0",
			"output1":[
				{"pdno":"005930","hldg_qty":"10","prdt_name":"삼성전자"},
				{"pdno":"000660","hldg_qty":"5","prdt_name":"SK하이닉스"}
			],
			"output2":[{"dnca_tot_amt":"1000000"}]
		}`)
	}))
	mgr := makeManager(t, "tok")
	bc := &DomesticBalanceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	snap, err := bc.FetchAccountSnapshot("12345678", "01")
	if err != nil {
		t.Fatalf("FetchAccountSnapshot: %v", err)
	}
	if snap.CashBalance.String() != "1000000" {
		t.Errorf("CashBalance = %v, want 1000000", snap.CashBalance)
	}
	if len(snap.Holdings) != 2 {
		t.Fatalf("holdings count = %d, want 2", len(snap.Holdings))
	}
	// sorted by ticker
	if snap.Holdings[0].Ticker != "000660" {
		t.Errorf("Holdings[0].Ticker = %q, want 000660", snap.Holdings[0].Ticker)
	}
}

func TestFetchAccountSnapshotPaginated(t *testing.T) {
	callCount := 0
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Header().Set("tr_cont", "M")
			fmt.Fprintf(w, `{
				"rt_cd":"0",
				"output1":[{"pdno":"005930","hldg_qty":"10","prdt_name":"삼성전자"}],
				"output2":[{"dnca_tot_amt":"500000"}],
				"ctx_area_fk100":"fk_value",
				"ctx_area_nk100":"nk_value"
			}`)
			return
		}
		// second page: no tr_cont header
		fmt.Fprintf(w, `{
			"rt_cd":"0",
			"output1":[{"pdno":"000660","hldg_qty":"5","prdt_name":"SK하이닉스"}],
			"output2":[{"dnca_tot_amt":"1000000"}]
		}`)
	}))
	mgr := makeManager(t, "tok")
	bc := &DomesticBalanceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	snap, err := bc.FetchAccountSnapshot("12345678", "01")
	if err != nil {
		t.Fatalf("FetchAccountSnapshot paginated: %v", err)
	}
	if len(snap.Holdings) != 2 {
		t.Fatalf("holdings count = %d, want 2", len(snap.Holdings))
	}
	if callCount != 2 {
		t.Errorf("page calls = %d, want 2", callCount)
	}
}

func TestFetchAccountSnapshotStopsWhenHeaderIsFinal(t *testing.T) {
	callCount := 0
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("tr_cont", "F")
		fmt.Fprintf(w, `{
			"rt_cd":"0",
			"output1":[{"pdno":"005930","hldg_qty":"10","prdt_name":"삼성전자"}],
			"output2":[{"dnca_tot_amt":"500000"}],
			"ctx_area_fk100":"fk_value",
			"ctx_area_nk100":"nk_value"
		}`)
	}))
	mgr := makeManager(t, "tok")
	bc := &DomesticBalanceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	if _, err := bc.FetchAccountSnapshot("12345678", "01"); err != nil {
		t.Fatalf("FetchAccountSnapshot: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("page calls = %d, want 1", callCount)
	}
}

// ---------- domestic_info.go ----------

func TestExtractOutputObject(t *testing.T) {
	// object form
	raw := map[string]json.RawMessage{"output": json.RawMessage(`{"pdno":"005930","prdt_name":"삼성전자","prdt_type_cd":"300"}`)}
	obj, err := extractOutputObject(raw)
	if err != nil {
		t.Fatalf("extractOutputObject object: %v", err)
	}
	if obj["pdno"] != "005930" {
		t.Errorf("pdno = %q, want 005930", obj["pdno"])
	}

	// array form
	raw2 := map[string]json.RawMessage{"output": json.RawMessage(`[{"pdno":"005930","prdt_name":"삼성전자"}]`)}
	obj2, err := extractOutputObject(raw2)
	if err != nil {
		t.Fatalf("extractOutputObject array: %v", err)
	}
	if obj2["pdno"] != "005930" {
		t.Errorf("pdno from array = %q, want 005930", obj2["pdno"])
	}

	// missing output field → error
	raw3 := map[string]json.RawMessage{"other": json.RawMessage(`{}`)}
	_, err = extractOutputObject(raw3)
	if err == nil {
		t.Error("expected error for missing output field")
	}
}

func TestDomesticInfoClientFetchBasicInfo(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"output":{"pdno":"005930","prdt_name":"삼성전자","prdt_type_cd":"300"}}`)
	}))
	mgr := makeManager(t, "tok")
	ic := &DomesticInfoClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", TrID: "CTPF1002R", CustType: "P",
		Manager: mgr,
	}
	info, err := ic.FetchBasicInfo("300", "005930")
	if err != nil {
		t.Fatalf("FetchBasicInfo: %v", err)
	}
	if info.Name != "삼성전자" {
		t.Errorf("Name = %q, want 삼성전자", info.Name)
	}
	if info.Pdno != "005930" {
		t.Errorf("Pdno = %q, want 005930", info.Pdno)
	}
}

func TestDomesticInfoClientFetchBasicInfoArrayOutput(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"output":[{"pdno":"005930","prdt_name":"삼성전자"}]}`)
	}))
	mgr := makeManager(t, "tok")
	ic := &DomesticInfoClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", TrID: "CTPF1002R", CustType: "P",
		Manager: mgr,
	}
	info, err := ic.FetchBasicInfo("300", "005930")
	if err != nil {
		t.Fatalf("FetchBasicInfo array: %v", err)
	}
	if info.Name != "삼성전자" {
		t.Errorf("Name = %q, want 삼성전자", info.Name)
	}
}

func TestDomesticInfoClientNameFallbacks(t *testing.T) {
	// prdt_name absent, prdt_name120 present
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"output":{"pdno":"005930","prdt_name120":"삼성전자120","prdt_type_cd":"300"}}`)
	}))
	mgr := makeManager(t, "tok")
	ic := &DomesticInfoClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", TrID: "CTPF1002R", CustType: "P",
		Manager: mgr,
	}
	info, err := ic.FetchBasicInfo("300", "005930")
	if err != nil {
		t.Fatalf("FetchBasicInfo name120: %v", err)
	}
	if info.Name != "삼성전자120" {
		t.Errorf("Name fallback prdt_name120 = %q, want 삼성전자120", info.Name)
	}
}

// ---------- domestic_order.go ----------

func TestDomesticOrderTrID(t *testing.T) {
	cases := []struct {
		side string
		demo bool
		want string
	}{
		{"buy", false, "TTTC0012U"},
		{"buy", true, "VTTC0012U"},
		{"sell", false, "TTTC0011U"},
		{"sell", true, "VTTC0011U"},
	}
	for _, tc := range cases {
		got := domesticOrderTrID(tc.side, tc.demo)
		if got != tc.want {
			t.Errorf("domesticOrderTrID(%q, %v) = %q, want %q", tc.side, tc.demo, got, tc.want)
		}
	}
}

func TestDomesticOrderClientPlaceOrderBuy(t *testing.T) {
	var received map[string]string
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode body: %v", err)
		}
		fmt.Fprintf(w, `{"rt_cd":"0","output":{"ORD_NO":"12345"}}`)
	}))
	mgr := makeManager(t, "tok")
	oc := &DomesticOrderClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s",
		CANO: "12345678", AcntPrdtCd: "01",
		CustType: "P", Env: "real",
		Manager: mgr,
	}
	result, err := oc.PlaceOrder("005930", "buy", 5, "")
	if err != nil {
		t.Fatalf("PlaceOrder buy: %v", err)
	}
	if result["rt_cd"] != "0" {
		t.Errorf("rt_cd = %v, want 0", result["rt_cd"])
	}
	if received["PDNO"] != "005930" {
		t.Errorf("PDNO = %q, want 005930", received["PDNO"])
	}
	if received["ORD_DVSN"] != "01" {
		t.Errorf("ORD_DVSN = %q, want 01", received["ORD_DVSN"])
	}
	if received["ORD_UNPR"] != "0" {
		t.Errorf("ORD_UNPR = %q, want 0", received["ORD_UNPR"])
	}
}

func TestDomesticOrderClientPlaceOrderSell(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"rt_cd":"0"}`)
	}))
	mgr := makeManager(t, "tok")
	oc := &DomesticOrderClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s",
		CANO: "12345678", AcntPrdtCd: "01",
		CustType: "P", Env: "real",
		Manager: mgr,
	}
	result, err := oc.PlaceOrder("005930", "sell", 3, "")
	if err != nil {
		t.Fatalf("PlaceOrder sell: %v", err)
	}
	if result["rt_cd"] != "0" {
		t.Errorf("rt_cd = %v, want 0", result["rt_cd"])
	}
}

func TestDomesticOrderClientHTTPError(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"server error"}`)
	}))
	mgr := makeManager(t, "tok")
	oc := &DomesticOrderClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s",
		CANO: "12345678", AcntPrdtCd: "01",
		CustType: "P", Env: "real",
		Manager: mgr,
	}
	_, err := oc.PlaceOrder("005930", "buy", 1, "")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// ---------- overseas_price.go ----------

func TestOverseasPriceClientFetchCurrentPrice(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"output":{"symbol":"AAPL","name":"Apple Inc.","last":"195.89"}}`)
	}))
	mgr := makeManager(t, "tok")
	pc := &OverseasPriceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	q, err := pc.FetchCurrentPrice("NASD", "AAPL")
	if err != nil {
		t.Fatalf("FetchCurrentPrice: %v", err)
	}
	if q.Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want AAPL", q.Symbol)
	}
	if q.Price != 195.89 {
		t.Errorf("Price = %v, want 195.89", q.Price)
	}
	if q.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", q.Currency)
	}
}

func TestOverseasPriceClientFetchHistoricalClose(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"output2":[{"xymd":"20240101","clos":"195.00"}]}`)
	}))
	mgr := makeManager(t, "tok")
	pc := &OverseasPriceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	price, err := pc.FetchHistoricalClose("NASD", "AAPL", datex.New(2024, 1, 1))
	if err != nil {
		t.Fatalf("FetchHistoricalClose: %v", err)
	}
	if price != 195.0 {
		t.Errorf("price = %v, want 195.0", price)
	}
}

func TestParseOverseasHistoricalExactMatch(t *testing.T) {
	data := []byte(`{"output2":[{"xymd":"20240101","clos":"195.00"},{"xymd":"20231231","clos":"194.00"}]}`)
	price := parseOverseasHistorical(data, "20240101")
	if price != 195.0 {
		t.Errorf("price = %v, want 195.0", price)
	}
}

func TestParseOverseasHistoricalFallback(t *testing.T) {
	data := []byte(`{"output2":[{"xymd":"20231231","clos":"194.00"}]}`)
	price := parseOverseasHistorical(data, "20240101")
	if price != 194.0 {
		t.Errorf("price = %v, want 194.0", price)
	}
}

func TestParseOverseasHistoricalEmpty(t *testing.T) {
	data := []byte(`{"output2":[]}`)
	price := parseOverseasHistorical(data, "20240101")
	if price != 0 {
		t.Errorf("price = %v, want 0", price)
	}
}

// TestOverseasPriceClientFetchCurrentPriceUsesShortEXCD verifies that FetchCurrentPrice
// sends the short KIS price-endpoint code (NAS/NYS/AMS) over the wire, even when the
// caller passes the canonical long code (NASD/NYSE/AMEX).
// The KIS price endpoint HHDFS00000300 requires NAS/NYS/AMS; the order endpoint uses NASD/NYSE/AMEX.
func TestOverseasPriceClientFetchCurrentPriceUsesShortEXCD(t *testing.T) {
	var gotEXCD string
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEXCD = r.URL.Query().Get("EXCD")
		fmt.Fprintf(w, `{"output":{"symbol":"AAPL","name":"Apple Inc.","last":"195.89"}}`)
	}))
	mgr := makeManager(t, "tok")
	pc := &OverseasPriceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	_, err := pc.FetchCurrentPrice("NASD", "AAPL")
	if err != nil {
		t.Fatalf("FetchCurrentPrice: %v", err)
	}
	if gotEXCD != "NAS" {
		t.Errorf("EXCD sent to price endpoint = %q, want NAS (price endpoint requires short codes)", gotEXCD)
	}
}

// TestOverseasPriceClientFetchHistoricalCloseUsesShortEXCD verifies that FetchHistoricalClose
// sends NAS/NYS/AMS (not NASD/NYSE/AMEX) to the KIS dailyprice endpoint HHDFS76240000.
func TestOverseasPriceClientFetchHistoricalCloseUsesShortEXCD(t *testing.T) {
	var gotEXCD string
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEXCD = r.URL.Query().Get("EXCD")
		fmt.Fprintf(w, `{"output2":[{"xymd":"20240101","clos":"195.00"}]}`)
	}))
	mgr := makeManager(t, "tok")
	pc := &OverseasPriceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	_, err := pc.FetchHistoricalClose("NASD", "AAPL", datex.New(2024, 1, 1))
	if err != nil {
		t.Fatalf("FetchHistoricalClose: %v", err)
	}
	if gotEXCD != "NAS" {
		t.Errorf("EXCD sent to dailyprice endpoint = %q, want NAS (price endpoint requires short codes)", gotEXCD)
	}
}

// ---------- overseas_info.go ----------

func TestOverseasInfoClientFetchBasicInfo(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"rt_cd":"0","output":{"pdno":"AAPL","prdt_name":"Apple Inc."}}`)
	}))
	mgr := makeManager(t, "tok")
	ic := &OverseasInfoClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", TrID: "CTPF1702R", CustType: "P",
		Manager: mgr,
	}
	info, err := ic.FetchBasicInfo("NAS", "AAPL")
	if err != nil {
		t.Fatalf("FetchBasicInfo: %v", err)
	}
	if info.Name != "Apple Inc." {
		t.Errorf("Name = %q, want Apple Inc.", info.Name)
	}
	if info.Pdno != "AAPL" {
		t.Errorf("Pdno = %q, want AAPL", info.Pdno)
	}
}

func TestOverseasInfoClientFetchBasicInfoBusinessError(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"rt_cd":"1","msg_cd":"ERR01","msg1":"ticker not found","output":{}}`)
	}))
	mgr := makeManager(t, "tok")
	ic := &OverseasInfoClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", TrID: "CTPF1702R", CustType: "P",
		Manager: mgr,
	}
	_, err := ic.FetchBasicInfo("NAS", "AAPL")
	if err == nil {
		t.Fatal("expected business error for rt_cd=1")
	}
}

func TestOverseasInfoClientNameFallbacks(t *testing.T) {
	// prdt_name absent, prdt_eng_name present
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"rt_cd":"0","output":{"pdno":"AAPL","prdt_eng_name":"Apple Incorporated"}}`)
	}))
	mgr := makeManager(t, "tok")
	ic := &OverseasInfoClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", TrID: "CTPF1702R", CustType: "P",
		Manager: mgr,
	}
	info, err := ic.FetchBasicInfo("NAS", "AAPL")
	if err != nil {
		t.Fatalf("FetchBasicInfo eng_name: %v", err)
	}
	if info.Name != "Apple Incorporated" {
		t.Errorf("Name fallback prdt_eng_name = %q, want Apple Incorporated", info.Name)
	}
}

// ---------- overseas_order.go ----------

func TestOverseasOrderTrID(t *testing.T) {
	cases := []struct {
		side string
		demo bool
		want string
	}{
		{"buy", false, "TTTT1002U"},
		{"buy", true, "VTTT1002U"},
		{"sell", false, "TTTT1006U"},
		{"sell", true, "VTTT1006U"},
	}
	for _, tc := range cases {
		got := overseasOrderTrID(tc.side, tc.demo)
		if got != tc.want {
			t.Errorf("overseasOrderTrID(%q, %v) = %q, want %q", tc.side, tc.demo, got, tc.want)
		}
	}
}

func TestOverseasOrderClientPlaceOrderBuy(t *testing.T) {
	var received map[string]string
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode body: %v", err)
		}
		fmt.Fprintf(w, `{"rt_cd":"0"}`)
	}))
	mgr := makeManager(t, "tok")
	oc := &OverseasOrderClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s",
		CustType: "P", Env: "real",
		Manager: mgr,
	}
	result, err := oc.PlaceOrder("AAPL", "buy", 2, "NASD")
	if err != nil {
		t.Fatalf("PlaceOrder buy: %v", err)
	}
	if result["rt_cd"] != "0" {
		t.Errorf("rt_cd = %v, want 0", result["rt_cd"])
	}
	if received["PDNO"] != "AAPL" {
		t.Errorf("PDNO = %q, want AAPL", received["PDNO"])
	}
	if received["OVRS_EXCG_CD"] != "NASD" {
		t.Errorf("OVRS_EXCG_CD = %q, want NASD", received["OVRS_EXCG_CD"])
	}
	if received["ORD_DVSN"] != "01" {
		t.Errorf("ORD_DVSN = %q, want 01", received["ORD_DVSN"])
	}
	if received["OVRS_ORD_UNPR"] != "0" {
		t.Errorf("OVRS_ORD_UNPR = %q, want 0", received["OVRS_ORD_UNPR"])
	}
}

func TestOverseasOrderClientPlaceOrderSell(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"rt_cd":"0"}`)
	}))
	mgr := makeManager(t, "tok")
	oc := &OverseasOrderClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s",
		CustType: "P", Env: "real",
		Manager: mgr,
	}
	result, err := oc.PlaceOrder("AAPL", "sell", 1, "NYSE")
	if err != nil {
		t.Fatalf("PlaceOrder sell: %v", err)
	}
	if result["rt_cd"] != "0" {
		t.Errorf("rt_cd = %v, want 0", result["rt_cd"])
	}
}

// ---------- unified_price.go ----------

func makeDomesticPriceHandler(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	})
}

func TestUnifiedPriceClientGetPriceDomestic(t *testing.T) {
	client, baseURL := makeClient(t, makeDomesticPriceHandler(
		`{"output":{"stck_code":"005930","hts_kor_isnm":"삼성전자","stck_prpr":"74000"}}`,
	))
	mgr := makeManager(t, "tok")
	domestic := &DomesticPriceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	upc := &UnifiedPriceClient{Domestic: domestic}
	q, err := upc.GetPrice("005930", "")
	if err != nil {
		t.Fatalf("GetPrice domestic: %v", err)
	}
	if q.Symbol != "005930" {
		t.Errorf("Symbol = %q, want 005930", q.Symbol)
	}
	if q.Price != 74000 {
		t.Errorf("Price = %v, want 74000", q.Price)
	}
}

func TestUnifiedPriceClientGetPriceOverseas(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"output":{"symbol":"AAPL","name":"Apple Inc.","last":"195.89"}}`)
	}))
	mgr := makeManager(t, "tok")
	overseas := &OverseasPriceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	upc := &UnifiedPriceClient{Overseas: overseas}
	q, err := upc.GetPrice("AAPL", "NASD")
	if err != nil {
		t.Fatalf("GetPrice overseas: %v", err)
	}
	if q.Price != 195.89 {
		t.Errorf("Price = %v, want 195.89", q.Price)
	}
}

func TestUnifiedPriceClientGetHistoricalCloseDomestic(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"output2":[{"stck_bsop_date":"20240101","stck_clpr":"73000"}]}`)
	}))
	mgr := makeManager(t, "tok")
	domestic := &DomesticPriceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	upc := &UnifiedPriceClient{Domestic: domestic}
	price, err := upc.GetHistoricalClose("005930", datex.New(2024, 1, 1), "")
	if err != nil {
		t.Fatalf("GetHistoricalClose domestic: %v", err)
	}
	if price != 73000.0 {
		t.Errorf("price = %v, want 73000.0", price)
	}
}

func TestUnifiedPriceClientGetHistoricalCloseOverseas(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"output2":[{"xymd":"20240101","clos":"195.00"}]}`)
	}))
	mgr := makeManager(t, "tok")
	overseas := &OverseasPriceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	upc := &UnifiedPriceClient{Overseas: overseas}
	price, err := upc.GetHistoricalClose("AAPL", datex.New(2024, 1, 1), "NASD")
	if err != nil {
		t.Fatalf("GetHistoricalClose overseas: %v", err)
	}
	if price != 195.0 {
		t.Errorf("price = %v, want 195.0", price)
	}
}

func TestUnifiedPriceClientGetDomesticPriceNameEnrichment(t *testing.T) {
	// price endpoint returns no name, info endpoint provides it
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "search-stock-info") {
			fmt.Fprintf(w, `{"output":{"pdno":"005930","prdt_name":"삼성전자","prdt_type_cd":"300"}}`)
			return
		}
		// inquire-price: no name
		fmt.Fprintf(w, `{"output":{"stck_code":"005930","hts_kor_isnm":"","stck_prpr":"74000"}}`)
	}))
	t.Cleanup(srv.Close)

	mgr := makeManager(t, "tok")
	domestic := &DomesticPriceClient{
		HTTP: srv.Client(), BaseURL: srv.URL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	info := &DomesticInfoClient{
		HTTP: srv.Client(), BaseURL: srv.URL,
		AppKey: "k", AppSecret: "s", TrID: "CTPF1002R", CustType: "P",
		Manager: mgr,
	}
	upc := &UnifiedPriceClient{Domestic: domestic, DomesticInfo: info, PrdtTypeCd: "300"}
	q, err := upc.GetPrice("005930", "")
	if err != nil {
		t.Fatalf("GetPrice with enrichment: %v", err)
	}
	if q.Name != "삼성전자" {
		t.Errorf("Name enriched = %q, want 삼성전자", q.Name)
	}
}

func TestUnifiedPriceClientGetOverseasPriceNoValidResponse(t *testing.T) {
	// All exchanges return HTTP 400 → fallback
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"not found"}`)
	}))
	mgr := makeManager(t, "tok")
	overseas := &OverseasPriceClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s", CustType: "P", Env: "real",
		Manager: mgr,
	}
	upc := &UnifiedPriceClient{Overseas: overseas}
	q, err := upc.GetPrice("AAPL", "")
	if err != nil {
		t.Fatalf("GetPrice fallback: %v", err)
	}
	if q.Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want AAPL", q.Symbol)
	}
	if q.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", q.Currency)
	}
}

func TestShortExchangeCode(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"NASD", "NAS"},
		{"NYSE", "NYS"},
		{"AMEX", "AMS"},
		{"OTHER", "OTHER"},
	}
	for _, tc := range cases {
		got := shortExchangeCode(tc.in)
		if got != tc.want {
			t.Errorf("shortExchangeCode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ---------- unified_order.go ----------

func TestNormalizeOrderExchange(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"NAS", "NASD"},
		{"NYS", "NYSE"},
		{"AMS", "AMEX"},
		{"", "NASD"},
		{"UNKNOWN", "UNKNOWN"},
		{"NASD", "NASD"}, // already normalized → passthrough
	}
	for _, tc := range cases {
		got := normalizeOrderExchange(tc.in)
		if got != tc.want {
			t.Errorf("normalizeOrderExchange(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestUnifiedOrderClientPlaceOrderDomestic(t *testing.T) {
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"rt_cd":"0"}`)
	}))
	mgr := makeManager(t, "tok")
	domestic := &DomesticOrderClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s",
		CANO: "12345678", AcntPrdtCd: "01",
		CustType: "P", Env: "real",
		Manager: mgr,
	}
	uc := &UnifiedOrderClient{Domestic: domestic}
	result, err := uc.PlaceOrder("005930", "buy", 5, "")
	if err != nil {
		t.Fatalf("PlaceOrder domestic: %v", err)
	}
	if result["rt_cd"] != "0" {
		t.Errorf("rt_cd = %v, want 0", result["rt_cd"])
	}
}

func TestUnifiedOrderClientPlaceOrderOverseas(t *testing.T) {
	var received map[string]string
	client, baseURL := makeClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode body: %v", err)
		}
		fmt.Fprintf(w, `{"rt_cd":"0"}`)
	}))
	mgr := makeManager(t, "tok")
	overseas := &OverseasOrderClient{
		HTTP: client, BaseURL: baseURL,
		AppKey: "k", AppSecret: "s",
		CustType: "P", Env: "real",
		Manager: mgr,
	}
	uc := &UnifiedOrderClient{Overseas: overseas}
	result, err := uc.PlaceOrder("AAPL", "buy", 2, "NAS")
	if err != nil {
		t.Fatalf("PlaceOrder overseas: %v", err)
	}
	if result["rt_cd"] != "0" {
		t.Errorf("rt_cd = %v, want 0", result["rt_cd"])
	}
	// NAS → normalized to NASD
	if received["OVRS_EXCG_CD"] != "NASD" {
		t.Errorf("OVRS_EXCG_CD = %q, want NASD", received["OVRS_EXCG_CD"])
	}
}

func TestUnifiedOrderClientPlaceOrderNilDomestic(t *testing.T) {
	uc := &UnifiedOrderClient{Domestic: nil}
	_, err := uc.PlaceOrder("005930", "buy", 1, "")
	if err == nil {
		t.Fatal("expected error for nil domestic client")
	}
}

func TestUnifiedOrderClientPlaceOrderNilOverseas(t *testing.T) {
	uc := &UnifiedOrderClient{Overseas: nil}
	_, err := uc.PlaceOrder("AAPL", "buy", 1, "NASD")
	if err == nil {
		t.Fatal("expected error for nil overseas client")
	}
}

// ---------- errors.go ----------

func TestKisAPIBusinessError(t *testing.T) {
	e := &KisAPIBusinessError{Code: "EBZ12345", Message: "some error"}
	want := "EBZ12345: some error"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
