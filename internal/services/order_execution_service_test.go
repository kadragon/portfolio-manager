package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

type fakeAccountLister struct {
	accounts []models.Account
	err      error
}

func (f *fakeAccountLister) ListAll(ctx context.Context) ([]models.Account, error) {
	return f.accounts, f.err
}

type executionCall struct {
	ticker, side, currency, status, message, exchange string
	quantity                                          int
	rawResponse                                       map[string]any
}

type fakeExecutionRecorder struct {
	calls []executionCall
	err   error
}

func (f *fakeExecutionRecorder) Create(
	ctx context.Context,
	ticker, side string,
	quantity int,
	currency, status, message string,
	exchange string,
	rawResponse map[string]any,
) (models.OrderExecutionRecord, error) {
	if f.err != nil {
		return models.OrderExecutionRecord{}, f.err
	}
	f.calls = append(f.calls, executionCall{ticker, side, currency, status, message, exchange, quantity, rawResponse})
	return models.OrderExecutionRecord{
		ID: uuidx.New(), Ticker: ticker, Side: side, Quantity: quantity,
		Currency: currency, Exchange: exchange, Status: status, Message: message, RawResponse: rawResponse,
	}, nil
}

type fakeKISOrderClient struct {
	resp        map[string]any
	err         error
	calledPrice string
}

func (f *fakeKISOrderClient) PlaceOrder(ctx context.Context, ticker, side string, quantity int, exchange, price string) (map[string]any, error) {
	f.calledPrice = price
	return f.resp, f.err
}

type fakeTossOrderClient struct {
	calledSeq    string
	calledIntent models.OrderIntent
	resp         map[string]any
	err          error
}

func (f *fakeTossOrderClient) PlaceOrder(ctx context.Context, accountSeq string, intent models.OrderIntent) (map[string]any, error) {
	f.calledSeq = accountSeq
	f.calledIntent = intent
	return f.resp, f.err
}

func ptrInt64OES(v int64) *int64 { return &v }
func ptrStr(v string) *string    { return &v }

func kisAccount(name string, keyID *int64) models.Account {
	return models.Account{ID: uuidx.New(), Name: name, KisAccountNo: ptrStr("12345678-01"), KisAPIKeyID: keyID}
}

func tossAccount(name string, seq int64) models.Account {
	return models.Account{ID: uuidx.New(), Name: name, TossAccountSeq: &seq}
}

func TestOrderExecutionService_PlaceOrder_RoutesKISWithParsedCANO(t *testing.T) {
	acct := kisAccount("ISA", ptrInt64OES(2))
	accounts := &fakeAccountLister{accounts: []models.Account{acct}}
	executions := &fakeExecutionRecorder{}

	var gotKeyID *int64
	var gotCano, gotAcntPrdtCd string
	kisClient := &fakeKISOrderClient{resp: map[string]any{"rt_cd": "0"}}
	factory := func(keyID *int64, cano, acntPrdtCd string) (KISOrderClient, error) {
		gotKeyID, gotCano, gotAcntPrdtCd = keyID, cano, acntPrdtCd
		return kisClient, nil
	}

	svc := NewOrderExecutionService(accounts, executions, factory, nil)
	record, err := svc.PlaceOrder(context.Background(), "ISA", "069500", "buy", 10, "", "KRW", "")
	if err != nil {
		t.Fatalf("PlaceOrder returned error: %v", err)
	}
	if gotKeyID == nil || *gotKeyID != 2 {
		t.Errorf("factory keyID = %v, want 2", gotKeyID)
	}
	if gotCano != "12345678" || gotAcntPrdtCd != "01" {
		t.Errorf("factory cano/acntPrdtCd = %q/%q, want 12345678/01", gotCano, gotAcntPrdtCd)
	}
	if record.Status != "success" {
		t.Errorf("record.Status = %q, want success", record.Status)
	}
	if len(executions.calls) != 1 {
		t.Fatalf("expected 1 execution recorded, got %d", len(executions.calls))
	}
}

func TestOrderExecutionService_PlaceOrder_RoutesToss(t *testing.T) {
	acct := tossAccount("TOSS", 42)
	accounts := &fakeAccountLister{accounts: []models.Account{acct}}
	executions := &fakeExecutionRecorder{}
	toss := &fakeTossOrderClient{resp: map[string]any{"result": map[string]any{"orderId": "abc"}}}

	svc := NewOrderExecutionService(accounts, executions, nil, toss)
	record, err := svc.PlaceOrder(context.Background(), "TOSS", "AAPL", "sell", 3, "", "USD", "")
	if err != nil {
		t.Fatalf("PlaceOrder returned error: %v", err)
	}
	if toss.calledSeq != "42" {
		t.Errorf("toss called with accountSeq = %q, want 42", toss.calledSeq)
	}
	if toss.calledIntent.Ticker != "AAPL" || toss.calledIntent.Side != "sell" || toss.calledIntent.Quantity != 3 {
		t.Errorf("toss intent = %+v, unexpected", toss.calledIntent)
	}
	if record.Status != "success" {
		t.Errorf("record.Status = %q, want success", record.Status)
	}
}

func TestOrderExecutionService_PlaceOrder_NormalizesTickerCase(t *testing.T) {
	acct := tossAccount("TOSS", 42)
	accounts := &fakeAccountLister{accounts: []models.Account{acct}}
	executions := &fakeExecutionRecorder{}
	toss := &fakeTossOrderClient{resp: map[string]any{"result": map[string]any{"orderId": "abc"}}}

	svc := NewOrderExecutionService(accounts, executions, nil, toss)
	if _, err := svc.PlaceOrder(context.Background(), "TOSS", "aapl", "sell", 3, "", "USD", ""); err != nil {
		t.Fatalf("PlaceOrder returned error: %v", err)
	}
	if toss.calledIntent.Ticker != "AAPL" {
		t.Errorf("toss intent ticker = %q, want AAPL", toss.calledIntent.Ticker)
	}
	if len(executions.calls) != 1 || executions.calls[0].ticker != "AAPL" {
		t.Fatalf("recorded execution ticker not normalized: %+v", executions.calls)
	}
}

func TestOrderExecutionService_PlaceOrder_UnknownAccount(t *testing.T) {
	accounts := &fakeAccountLister{accounts: []models.Account{kisAccount("ISA", nil)}}
	svc := NewOrderExecutionService(accounts, &fakeExecutionRecorder{}, nil, nil)
	_, err := svc.PlaceOrder(context.Background(), "NOPE", "069500", "buy", 1, "", "KRW", "")
	if err == nil {
		t.Fatal("expected error for unknown account, got nil")
	}
}

func TestOrderExecutionService_PlaceOrder_AmbiguousAccount(t *testing.T) {
	accounts := &fakeAccountLister{accounts: []models.Account{
		kisAccount("ISA 연금", nil),
		kisAccount("ISA 중개형", nil),
	}}
	svc := NewOrderExecutionService(accounts, &fakeExecutionRecorder{}, nil, nil)
	_, err := svc.PlaceOrder(context.Background(), "ISA", "069500", "buy", 1, "", "KRW", "")
	if err == nil {
		t.Fatal("expected ambiguous-match error, got nil")
	}
}

func TestOrderExecutionService_PlaceOrder_NoBrokerLinked(t *testing.T) {
	accounts := &fakeAccountLister{accounts: []models.Account{{ID: uuidx.New(), Name: "연금저축"}}}
	svc := NewOrderExecutionService(accounts, &fakeExecutionRecorder{}, nil, nil)
	_, err := svc.PlaceOrder(context.Background(), "연금저축", "069500", "buy", 1, "", "KRW", "")
	if err == nil {
		t.Fatal("expected error for account with no broker link, got nil")
	}
}

func TestOrderExecutionService_PlaceOrder_KISFailureStillRecorded(t *testing.T) {
	acct := kisAccount("ISA", nil)
	accounts := &fakeAccountLister{accounts: []models.Account{acct}}
	executions := &fakeExecutionRecorder{}
	kisClient := &fakeKISOrderClient{resp: map[string]any{"rt_cd": "1", "msg1": "잔고부족"}}
	factory := func(keyID *int64, cano, acntPrdtCd string) (KISOrderClient, error) { return kisClient, nil }

	svc := NewOrderExecutionService(accounts, executions, factory, nil)
	record, err := svc.PlaceOrder(context.Background(), "ISA", "069500", "buy", 10, "", "KRW", "")
	if err != nil {
		t.Fatalf("PlaceOrder returned error: %v", err)
	}
	if record.Status != "failed" || record.Message != "잔고부족" {
		t.Errorf("record = %+v, want status=failed message=잔고부족", record)
	}
}

func TestOrderExecutionService_PlaceOrder_TransportErrorStillRecorded(t *testing.T) {
	acct := kisAccount("ISA", nil)
	accounts := &fakeAccountLister{accounts: []models.Account{acct}}
	executions := &fakeExecutionRecorder{}
	kisClient := &fakeKISOrderClient{err: errors.New("network timeout")}
	factory := func(keyID *int64, cano, acntPrdtCd string) (KISOrderClient, error) { return kisClient, nil }

	svc := NewOrderExecutionService(accounts, executions, factory, nil)
	record, err := svc.PlaceOrder(context.Background(), "ISA", "069500", "buy", 10, "", "KRW", "")
	if err != nil {
		t.Fatalf("PlaceOrder returned error: %v", err)
	}
	if record.Status != "failed" || record.Message != "network timeout" {
		t.Errorf("record = %+v, want status=failed message=network timeout", record)
	}
}

func TestOrderExecutionService_PlaceOrder_InvalidSideOrQuantity(t *testing.T) {
	svc := NewOrderExecutionService(&fakeAccountLister{}, &fakeExecutionRecorder{}, nil, nil)
	if _, err := svc.PlaceOrder(context.Background(), "ISA", "069500", "hold", 1, "", "KRW", ""); err == nil {
		t.Error("expected error for invalid side")
	}
	if _, err := svc.PlaceOrder(context.Background(), "ISA", "069500", "buy", 0, "", "KRW", ""); err == nil {
		t.Error("expected error for non-positive quantity")
	}
}

func TestOrderExecutionService_PlaceOrder_EmptyAccountNameRejected(t *testing.T) {
	// An empty account name must never resolve — substring matching would
	// otherwise treat every account name as "containing" "" and pick one
	// (or error ambiguous) essentially at random.
	accounts := &fakeAccountLister{accounts: []models.Account{kisAccount("ISA", nil)}}
	svc := NewOrderExecutionService(accounts, &fakeExecutionRecorder{}, nil, nil)
	if _, err := svc.PlaceOrder(context.Background(), "   ", "069500", "buy", 1, "", "KRW", ""); err == nil {
		t.Fatal("expected error for blank account name, got nil")
	}
}

func TestOrderExecutionService_PlaceOrder_UnlinkedAccountNeverRoutesToKIS(t *testing.T) {
	// Regression guard: an account with neither TossAccountSeq nor
	// KisAccountNo must always hit the "not linked" error, never a KIS
	// client — there is no "default account" fallback to silently route
	// through. A nil kisFactory here means any accidental KIS-branch call
	// panics instead of passing quietly.
	accounts := &fakeAccountLister{accounts: []models.Account{{ID: uuidx.New(), Name: "연금저축", KisAPIKeyID: ptrInt64OES(1)}}}
	svc := NewOrderExecutionService(accounts, &fakeExecutionRecorder{}, nil, nil)
	_, err := svc.PlaceOrder(context.Background(), "연금저축", "069500", "buy", 1, "", "KRW", "")
	if err == nil {
		t.Fatal("expected 'not linked to KIS or Toss' error, got nil")
	}
}

func TestOrderExecutionService_PlaceOrder_PersistFailureStillReportsOutcome(t *testing.T) {
	acct := kisAccount("ISA", nil)
	accounts := &fakeAccountLister{accounts: []models.Account{acct}}
	executions := &fakeExecutionRecorder{err: errors.New("db locked")}
	kisClient := &fakeKISOrderClient{resp: map[string]any{"rt_cd": "0"}}
	factory := func(keyID *int64, cano, acntPrdtCd string) (KISOrderClient, error) { return kisClient, nil }

	svc := NewOrderExecutionService(accounts, executions, factory, nil)
	record, err := svc.PlaceOrder(context.Background(), "ISA", "069500", "buy", 10, "", "KRW", "")
	if err != nil {
		t.Fatalf("PlaceOrder returned error: %v (order succeeded — losing that fact risks a duplicate resubmission)", err)
	}
	if record.Status != "success" {
		t.Errorf("record.Status = %q, want success (the order itself succeeded)", record.Status)
	}
	if !strings.Contains(record.Message, "db locked") {
		t.Errorf("record.Message = %q, want it to mention the persist failure", record.Message)
	}
}

func TestOrderExecutionService_PlaceOrder_LimitPriceReachesKIS(t *testing.T) {
	acct := kisAccount("ISA", nil)
	accounts := &fakeAccountLister{accounts: []models.Account{acct}}
	kisClient := &fakeKISOrderClient{resp: map[string]any{"rt_cd": "0"}}
	factory := func(keyID *int64, cano, acntPrdtCd string) (KISOrderClient, error) { return kisClient, nil }

	svc := NewOrderExecutionService(accounts, &fakeExecutionRecorder{}, factory, nil)
	if _, err := svc.PlaceOrder(context.Background(), "ISA", "069500", "buy", 10, "", "KRW", "27470"); err != nil {
		t.Fatalf("PlaceOrder returned error: %v", err)
	}
	if kisClient.calledPrice != "27470" {
		t.Errorf("KIS client received price = %q, want 27470", kisClient.calledPrice)
	}
}

func TestOrderExecutionService_PlaceOrder_InvalidLimitPriceRejected(t *testing.T) {
	// A malformed or non-positive -price must be rejected before any broker
	// call, so a typo can't be spent as a live order attempt. A nil kisFactory
	// means any accidental broker-branch call would panic instead of passing.
	acct := kisAccount("ISA", nil)
	accounts := &fakeAccountLister{accounts: []models.Account{acct}}
	svc := NewOrderExecutionService(accounts, &fakeExecutionRecorder{}, nil, nil)
	for _, bad := range []string{"abc", "0", "-5", "1O0"} {
		if _, err := svc.PlaceOrder(context.Background(), "ISA", "069500", "buy", 10, "", "KRW", bad); err == nil {
			t.Errorf("expected error for invalid price %q, got nil", bad)
		}
	}
}

func TestOrderExecutionService_PlaceOrder_LimitPriceRejectedForToss(t *testing.T) {
	acct := tossAccount("TOSS", 42)
	accounts := &fakeAccountLister{accounts: []models.Account{acct}}
	toss := &fakeTossOrderClient{resp: map[string]any{"result": map[string]any{"orderId": "abc"}}}

	svc := NewOrderExecutionService(accounts, &fakeExecutionRecorder{}, nil, toss)
	_, err := svc.PlaceOrder(context.Background(), "TOSS", "AAPL", "buy", 1, "", "USD", "150.00")
	if err == nil {
		t.Fatal("expected error for limit order on Toss account, got nil")
	}
	if toss.calledSeq != "" {
		t.Errorf("Toss client should not have been called for a rejected limit order, got seq %q", toss.calledSeq)
	}
}

func TestNormalizeKisAccountNo(t *testing.T) {
	cases := []struct {
		in, wantCano, wantAcntPrdtCd string
		wantErr                      bool
	}{
		{"12345678-01", "12345678", "01", false},
		{"1234567890", "12345678", "90", false},
		{"1234567", "", "", true},
	}
	for _, tc := range cases {
		cano, acntPrdtCd, err := normalizeKisAccountNo(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("normalizeKisAccountNo(%q) expected error, got none", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizeKisAccountNo(%q) unexpected error: %v", tc.in, err)
		}
		if cano != tc.wantCano || acntPrdtCd != tc.wantAcntPrdtCd {
			t.Errorf("normalizeKisAccountNo(%q) = %q/%q, want %q/%q", tc.in, cano, acntPrdtCd, tc.wantCano, tc.wantAcntPrdtCd)
		}
	}
}
