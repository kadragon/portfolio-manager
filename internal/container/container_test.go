package container

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/kis"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/services"
)

func ptrInt64(v int64) *int64 { return &v }

func TestResolveSyncService(t *testing.T) {
	defaultSync := &services.KisAccountSyncService{}
	key2 := &services.KisAccountSyncService{}
	byKeyID := map[int64]*services.KisAccountSyncService{2: key2}

	cases := []struct {
		name    string
		keyID   *int64
		want    *services.KisAccountSyncService
		wantLog bool
	}{
		{"nil keyID falls back to default", nil, defaultSync, false},
		{"keyID found returns mapped service", ptrInt64(2), key2, false},
		{"keyID 1 not found falls back silently", ptrInt64(1), defaultSync, false},
		{"keyID not 1 and not found warns", ptrInt64(3), defaultSync, true},
	}

	// No t.Parallel() — subtests redirect the global log.Writer; concurrent
	// execution would race on the shared buffer and restoration.
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			origOut, origFlags := log.Writer(), log.Flags()
			log.SetOutput(&buf)
			log.SetFlags(0)
			t.Cleanup(func() {
				log.SetOutput(origOut)
				log.SetFlags(origFlags)
			})

			got := resolveSyncService(defaultSync, byKeyID, tc.keyID)
			if got != tc.want {
				t.Errorf("resolveSyncService = %p, want %p", got, tc.want)
			}

			logged := strings.Contains(buf.String(), "no sync service for requested KIS key")
			if logged != tc.wantLog {
				t.Errorf("warning logged = %v (%q), want %v", logged, buf.String(), tc.wantLog)
			}

			// PR #111: the fallback warning must never leak the requested key ID value.
			if tc.wantLog && tc.keyID != nil {
				if keyStr := fmt.Sprintf("%d", *tc.keyID); strings.Contains(buf.String(), keyStr) {
					t.Errorf("warning log leaked key ID %q: %q", keyStr, buf.String())
				}
			}
		})
	}
}

func setKISEnv(t *testing.T) {
	t.Helper()
	t.Setenv("KIS_APP_KEY", "app-key")
	t.Setenv("KIS_APP_SECRET", "app-secret")
	t.Setenv("KIS_ENV", "demo")
	t.Setenv("KIS_CANO", "12345678")
	t.Setenv("KIS_ACNT_PRDT_CD", "01")
	t.Setenv("KIS_CUST_TYPE", "P")
}

func TestKISClientsShareTokenManager(t *testing.T) {
	setKISEnv(t)

	auth := buildKISAuth()
	if auth == nil {
		t.Fatal("expected KIS auth config")
	}
	priceClient, ok := buildKISClient(auth).(*kis.UnifiedPriceClient)
	if !ok {
		t.Fatal("expected unified price client")
	}
	orderClient := buildOrderClient(auth)
	if orderClient == nil {
		t.Fatal("expected unified order client")
	}
	balanceClient, ok := buildBalanceClient(auth).(*kis.DomesticBalanceClient)
	if !ok {
		t.Fatal("expected domestic balance client")
	}

	manager := priceClient.Domestic.Manager
	if manager == nil {
		t.Fatal("price manager is nil")
	}
	if priceClient.Overseas.Manager != manager ||
		priceClient.DomesticInfo.Manager != manager ||
		priceClient.OverseasInfo.Manager != manager ||
		orderClient.Domestic.Manager != manager ||
		orderClient.Overseas.Manager != manager ||
		balanceClient.Manager != manager {
		t.Fatal("KIS clients do not share one token manager")
	}
}

func TestContainerWiresAccountSyncIntoRebalanceExecution(t *testing.T) {
	setKISEnv(t)
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	c := newWithQueries(sqlDB, q, true)
	if c.AccountSync == nil {
		t.Fatal("account sync is not configured")
	}
	field := reflect.ValueOf(c.RebalanceExecution).Elem().FieldByName("syncService")
	if field.IsNil() {
		t.Fatal("rebalance execution sync service is nil")
	}
}

type fakeKISOrderClient struct {
	calls []models.OrderIntent
}

func (f *fakeKISOrderClient) PlaceOrder(ticker, side string, quantity int, exchange string) (map[string]any, error) {
	f.calls = append(f.calls, models.OrderIntent{
		Ticker:   ticker,
		Side:     side,
		Quantity: quantity,
		Exchange: exchange,
	})
	return map[string]any{"broker": "kis"}, nil
}

type fakeTossOrderClient struct {
	accountSeqs []string
	intents     []models.OrderIntent
}

func (f *fakeTossOrderClient) PlaceOrder(accountSeq string, intent models.OrderIntent) (map[string]any, error) {
	f.accountSeqs = append(f.accountSeqs, accountSeq)
	f.intents = append(f.intents, intent)
	return map[string]any{"broker": "toss"}, nil
}

func TestAccountOrderRouterUsesTossAccountSeq(t *testing.T) {
	ctx := context.Background()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	c := NewWithQueries(sqlDB, q)

	account, err := c.Accounts.Create(ctx, "Toss", numeric.Zero)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	account, err = c.Accounts.Update(ctx, account.ID, account.Name, account.CashBalance,
		sql.NullString{}, sql.NullInt64{}, sql.NullString{}, sql.NullInt64{Int64: 7, Valid: true})
	if err != nil {
		t.Fatalf("set toss account seq: %v", err)
	}

	kisClient := &fakeKISOrderClient{}
	tossClient := &fakeTossOrderClient{}
	router := &accountOrderRouter{accounts: c.Accounts, kis: kisClient, toss: tossClient}

	resp, err := router.PlaceOrder(models.OrderIntent{
		AccountID: account.ID,
		Ticker:    "005930",
		Side:      "buy",
		Quantity:  3,
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if resp["broker"] != "toss" {
		t.Fatalf("broker = %v, want toss", resp["broker"])
	}
	if len(tossClient.accountSeqs) != 1 || tossClient.accountSeqs[0] != "7" {
		t.Fatalf("toss account seqs = %v, want [7]", tossClient.accountSeqs)
	}
	if len(kisClient.calls) != 0 {
		t.Fatalf("KIS should not be called: %v", kisClient.calls)
	}
}

func TestAccountOrderRouterFallsBackToKIS(t *testing.T) {
	ctx := context.Background()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	c := NewWithQueries(sqlDB, q)

	account, err := c.Accounts.Create(ctx, "KIS", numeric.Zero)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	kisClient := &fakeKISOrderClient{}
	tossClient := &fakeTossOrderClient{}
	router := &accountOrderRouter{accounts: c.Accounts, kis: kisClient, toss: tossClient}

	resp, err := router.PlaceOrder(models.OrderIntent{
		AccountID: account.ID,
		Ticker:    "AAPL",
		Side:      "sell",
		Quantity:  2,
		Exchange:  "NASD",
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if resp["broker"] != "kis" {
		t.Fatalf("broker = %v, want kis", resp["broker"])
	}
	if len(kisClient.calls) != 1 || kisClient.calls[0].Ticker != "AAPL" {
		t.Fatalf("KIS calls = %v, want AAPL call", kisClient.calls)
	}
	if len(tossClient.intents) != 0 {
		t.Fatalf("Toss should not be called: %v", tossClient.intents)
	}
}

// --- kisAssetClassifier overseas exchange routing (P2-2) ---

func TestOverseasPriceEXCD(t *testing.T) {
	cases := []struct {
		in    string
		want  string
		known bool
	}{
		{"NASD", "NAS", true},
		{"NAS", "NAS", true},
		{"NYSE", "NYS", true},
		{"nys", "NYS", true},
		{"AMEX", "AMS", true},
		{" ams ", "AMS", true},
		{"", "", false},
		{"TSE", "", false},
	}
	for _, c := range cases {
		got, ok := overseasPriceEXCD(c.in)
		if got != c.want || ok != c.known {
			t.Errorf("overseasPriceEXCD(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.known)
		}
	}
}

type fakeOverseasInfo struct {
	// byEXCD maps an exchange code to the asset class resolved there; a missing
	// entry simulates a KIS "wrong market" business error.
	byEXCD map[string]string
	calls  []string
}

func (f *fakeOverseasInfo) ClassifyAssetClass(excd, ticker string) (string, error) {
	f.calls = append(f.calls, excd)
	if ac, ok := f.byEXCD[excd]; ok {
		return ac, nil
	}
	return "", fmt.Errorf("not listed on %s", excd)
}

func TestKisAssetClassifierKnownExchangeNoFallback(t *testing.T) {
	fake := &fakeOverseasInfo{byEXCD: map[string]string{"NYS": "etf"}}
	k := &kisAssetClassifier{overseas: fake}

	ac, err := k.ClassifyAssetClass("SCHD", "NYSE")
	if err != nil || ac != "etf" {
		t.Fatalf("ClassifyAssetClass = (%q,%v), want (etf,nil)", ac, err)
	}
	if len(fake.calls) != 1 || fake.calls[0] != "NYS" {
		t.Errorf("calls = %v, want single [NYS] (no fallback for a known exchange)", fake.calls)
	}
}

func TestKisAssetClassifierUnknownExchangeFallsBack(t *testing.T) {
	// Resolvable only on AMS; an unknown exchange must try NAS, NYS, then AMS.
	fake := &fakeOverseasInfo{byEXCD: map[string]string{"AMS": "stock"}}
	k := &kisAssetClassifier{overseas: fake}

	ac, err := k.ClassifyAssetClass("XYZ", "")
	if err != nil || ac != "stock" {
		t.Fatalf("ClassifyAssetClass = (%q,%v), want (stock,nil)", ac, err)
	}
	want := []string{"NAS", "NYS", "AMS"}
	if !reflect.DeepEqual(fake.calls, want) {
		t.Errorf("calls = %v, want %v", fake.calls, want)
	}
}

func TestKisAssetClassifierUnknownExchangeAllFailReturnsLastErr(t *testing.T) {
	fake := &fakeOverseasInfo{byEXCD: map[string]string{}}
	k := &kisAssetClassifier{overseas: fake}

	if _, err := k.ClassifyAssetClass("XYZ", "BOGUS"); err == nil {
		t.Fatal("want error when no exchange resolves")
	}
	if len(fake.calls) != 3 {
		t.Errorf("calls = %v, want all 3 exchanges tried", fake.calls)
	}
}
