package kis

import (
	"testing"
)

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
