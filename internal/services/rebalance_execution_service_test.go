package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/services"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// --- mock helpers ---

type mockOrderClient struct {
	calls    []mockOrderCall
	response map[string]any
	err      error
	perCall  []func() (map[string]any, error)
}

type mockOrderCall struct {
	intent models.OrderIntent
}

func (m *mockOrderClient) PlaceOrder(intent models.OrderIntent) (map[string]any, error) {
	m.calls = append(m.calls, mockOrderCall{intent: intent})
	if len(m.perCall) > 0 {
		fn := m.perCall[0]
		m.perCall = m.perCall[1:]
		return fn()
	}
	return m.response, m.err
}

type mockExecRepo struct {
	calls []mockCreateCall
}

type mockCreateCall struct {
	ticker, side    string
	quantity        int
	currency        string
	status, message string
	exchange        string
}

func (r *mockExecRepo) Create(_ context.Context, ticker, side string, qty int, currency, status, message, exchange string, _ map[string]any) (models.OrderExecutionRecord, error) {
	r.calls = append(r.calls, mockCreateCall{ticker, side, qty, currency, status, message, exchange})
	return models.OrderExecutionRecord{}, nil
}

type mockSyncService struct {
	calls int
	err   error
}

func (s *mockSyncService) SyncAccount() error {
	s.calls++
	return s.err
}

// --- helpers ---

func makeRec(ticker string, action models.RebalanceAction, currency string, qty string) models.RebalanceRecommendation {
	rec := models.RebalanceRecommendation{
		Ticker:   ticker,
		Action:   action,
		Amount:   mustN("100"),
		Priority: 1,
		Currency: currency,
	}
	if qty != "" {
		q := mustN(qty)
		rec.Quantity = &q
	}
	return rec
}

// --- tests ---

func TestCreateOrderIntentsSellBeforeBuy(t *testing.T) {
	recs := []models.RebalanceRecommendation{
		makeRec("005930", models.ActionBuy, "KRW", "10"),
		makeRec("AAPL", models.ActionSell, "USD", "3"),
		makeRec("MSFT", models.ActionBuy, "USD", "1"),
		makeRec("000660", models.ActionSell, "KRW", "2"),
	}

	svc := services.NewRebalanceExecutionService(nil, nil, nil)
	result := svc.CreateOrderIntents(recs, nil)

	if len(result.Intents) != 4 {
		t.Fatalf("want 4 intents, got %d", len(result.Intents))
	}
	if result.Intents[0].Side != "sell" || result.Intents[1].Side != "sell" {
		t.Errorf("first two should be sells: %v %v", result.Intents[0].Side, result.Intents[1].Side)
	}
	if result.Intents[2].Side != "buy" || result.Intents[3].Side != "buy" {
		t.Errorf("last two should be buys: %v %v", result.Intents[2].Side, result.Intents[3].Side)
	}
}

func TestCreateOrderIntentsFloorQuantity(t *testing.T) {
	recs := []models.RebalanceRecommendation{
		makeRec("005930", models.ActionBuy, "KRW", "7.9"),
		makeRec("AAPL", models.ActionSell, "USD", "3.1"),
	}

	svc := services.NewRebalanceExecutionService(nil, nil, nil)
	result := svc.CreateOrderIntents(recs, nil)

	if len(result.Intents) != 2 {
		t.Fatalf("want 2 intents, got %d", len(result.Intents))
	}
	// sells first
	sell := result.Intents[0]
	buy := result.Intents[1]
	if sell.Quantity != 3 {
		t.Errorf("AAPL qty: want 3, got %d", sell.Quantity)
	}
	if buy.Quantity != 7 {
		t.Errorf("005930 qty: want 7, got %d", buy.Quantity)
	}
}

func TestCreateOrderIntentsZeroQtySkipped(t *testing.T) {
	recs := []models.RebalanceRecommendation{
		makeRec("005930", models.ActionBuy, "KRW", "5"),
		makeRec("AAPL", models.ActionBuy, "USD", "0.3"), // floor→0
		makeRec("TSLA", models.ActionSell, "USD", ""),   // nil qty→0
	}

	svc := services.NewRebalanceExecutionService(nil, nil, nil)
	result := svc.CreateOrderIntents(recs, nil)

	if len(result.Intents) != 1 {
		t.Fatalf("want 1 intent, got %d", len(result.Intents))
	}
	if result.Intents[0].Ticker != "005930" {
		t.Errorf("want 005930, got %q", result.Intents[0].Ticker)
	}
	if len(result.Skipped) != 2 {
		t.Fatalf("want 2 skipped, got %d", len(result.Skipped))
	}
}

func TestCreateOrderIntentsOverseasExchange(t *testing.T) {
	recs := []models.RebalanceRecommendation{
		makeRec("AAPL", models.ActionSell, "USD", "2"),
		makeRec("MSFT", models.ActionBuy, "USD", "1"),
		makeRec("005930", models.ActionBuy, "KRW", "10"),
	}
	exchangeMap := map[string]string{"AAPL": "NYSE"}

	svc := services.NewRebalanceExecutionService(nil, nil, nil)
	result := svc.CreateOrderIntents(recs, exchangeMap)

	byTicker := map[string]models.OrderIntent{}
	for _, in := range result.Intents {
		byTicker[in.Ticker] = in
	}

	if byTicker["AAPL"].Exchange != "NYSE" {
		t.Errorf("AAPL exchange: want NYSE, got %q", byTicker["AAPL"].Exchange)
	}
	if byTicker["MSFT"].Exchange != "NASD" {
		t.Errorf("MSFT exchange: want NASD, got %q", byTicker["MSFT"].Exchange)
	}
	if byTicker["005930"].Exchange != "" {
		t.Errorf("005930 exchange: want empty, got %q", byTicker["005930"].Exchange)
	}
}

func TestCreateOrderIntentsPreservesAccountAndAmount(t *testing.T) {
	accountID := uuidx.New()
	rec := makeRec("005930", models.ActionBuy, "KRW", "7")
	rec.AccountID = accountID
	rec.AccountName = "Toss"
	rec.Amount = mustN("700000")

	svc := services.NewRebalanceExecutionService(nil, nil, nil)
	result := svc.CreateOrderIntents([]models.RebalanceRecommendation{rec}, nil)

	if len(result.Intents) != 1 {
		t.Fatalf("want 1 intent, got %d", len(result.Intents))
	}
	got := result.Intents[0]
	if got.AccountID != accountID || got.AccountName != "Toss" {
		t.Fatalf("account not preserved: %+v", got)
	}
	if !got.Amount.Equal(mustN("700000").Decimal) {
		t.Fatalf("amount = %s, want 700000", got.Amount.String())
	}
}

func TestExecuteDryRunNoAPICalls(t *testing.T) {
	recs := []models.RebalanceRecommendation{
		makeRec("005930", models.ActionBuy, "KRW", "7"),
		makeRec("AAPL", models.ActionSell, "USD", "2"),
	}

	client := &mockOrderClient{}
	svc := services.NewRebalanceExecutionService(client, nil, nil)
	result := svc.ExecuteRebalanceOrders(recs, true, nil)

	if len(client.calls) != 0 {
		t.Errorf("dry run: want 0 API calls, got %d", len(client.calls))
	}
	if len(result.Intents) != 2 {
		t.Errorf("want 2 intents, got %d", len(result.Intents))
	}
	if result.Intents[0].Side != "sell" {
		t.Errorf("first intent should be sell")
	}
}

func TestExecuteDefersBuysWhenSellsPresent(t *testing.T) {
	recs := []models.RebalanceRecommendation{
		makeRec("005930", models.ActionBuy, "KRW", "7"),
		makeRec("AAPL", models.ActionSell, "USD", "2"),
	}

	resp := map[string]any{"rt_cd": "0", "msg_cd": "APBK0013", "msg1": "주문 전송 완료"}
	client := &mockOrderClient{response: resp}
	svc := services.NewRebalanceExecutionService(client, nil, nil)
	result := svc.ExecuteRebalanceOrders(recs, false, nil)

	if len(client.calls) != 1 {
		t.Fatalf("want 1 API call, got %d", len(client.calls))
	}
	if client.calls[0].intent.Ticker != "AAPL" || client.calls[0].intent.Side != "sell" {
		t.Errorf("first call should be AAPL sell, got %+v", client.calls[0])
	}
	if len(result.Executions) != 1 {
		t.Fatalf("want 1 execution, got %d", len(result.Executions))
	}
	if result.Executions[0].Status != "success" {
		t.Errorf("sell should succeed")
	}
	if len(result.Deferred) != 1 || result.Deferred[0].Ticker != "005930" {
		t.Fatalf("deferred buys = %+v, want 005930", result.Deferred)
	}
}

func TestExecuteBuyOnlyCallsOrderAPI(t *testing.T) {
	recs := []models.RebalanceRecommendation{
		makeRec("005930", models.ActionBuy, "KRW", "7"),
		makeRec("000660", models.ActionBuy, "KRW", "3"),
	}

	resp := map[string]any{"rt_cd": "0", "msg_cd": "APBK0013", "msg1": "주문 전송 완료"}
	client := &mockOrderClient{response: resp}
	svc := services.NewRebalanceExecutionService(client, nil, nil)
	result := svc.ExecuteRebalanceOrders(recs, false, nil)

	if len(client.calls) != 2 {
		t.Fatalf("want 2 API calls, got %d", len(client.calls))
	}
	if client.calls[0].intent.Ticker != "005930" || client.calls[1].intent.Ticker != "000660" {
		t.Fatalf("calls = %+v, want buy rec order", client.calls)
	}
	if len(result.Deferred) != 0 {
		t.Fatalf("buy-only run should not defer orders: %+v", result.Deferred)
	}
}

func TestExecuteDefersSameAccountBuysOnly(t *testing.T) {
	accountA := uuidx.New()
	accountB := uuidx.New()

	recSellA := makeRec("AAPL", models.ActionSell, "USD", "2")
	recSellA.AccountID = accountA
	recBuyA := makeRec("MSFT", models.ActionBuy, "USD", "3")
	recBuyA.AccountID = accountA
	recBuyB := makeRec("005930", models.ActionBuy, "KRW", "7")
	recBuyB.AccountID = accountB

	resp := map[string]any{"rt_cd": "0", "msg_cd": "APBK0013", "msg1": "주문 전송 완료"}
	client := &mockOrderClient{response: resp}
	svc := services.NewRebalanceExecutionService(client, nil, nil)
	result := svc.ExecuteRebalanceOrders([]models.RebalanceRecommendation{recSellA, recBuyA, recBuyB}, false, nil)

	// Account A's buy is deferred (same account as sell), Account B's buy executes.
	if len(result.Deferred) != 1 || result.Deferred[0].Ticker != "MSFT" {
		t.Fatalf("deferred = %+v, want only MSFT (account A)", result.Deferred)
	}
	tickers := make(map[string]bool)
	for _, c := range client.calls {
		tickers[c.intent.Ticker] = true
	}
	if !tickers["AAPL"] || !tickers["005930"] {
		t.Fatalf("expected AAPL sell + 005930 buy to execute, got calls = %+v", client.calls)
	}
	if tickers["MSFT"] {
		t.Fatal("MSFT (account A buy) should not execute while account A has sell")
	}
}

func TestExecuteSingleFailureContinues(t *testing.T) {
	recs := []models.RebalanceRecommendation{
		makeRec("AAPL", models.ActionSell, "USD", "2"),
		makeRec("MSFT", models.ActionSell, "USD", "1"),
		makeRec("005930", models.ActionBuy, "KRW", "7"),
	}

	failResp := map[string]any{"rt_cd": "1", "msg_cd": "APBK0919", "msg1": "주문가능수량 부족"}
	okResp := map[string]any{"rt_cd": "0", "msg_cd": "APBK0013", "msg1": "주문 전송 완료"}

	client := &mockOrderClient{
		perCall: []func() (map[string]any, error){
			func() (map[string]any, error) { return okResp, nil },
			func() (map[string]any, error) { return failResp, nil },
		},
	}
	svc := services.NewRebalanceExecutionService(client, nil, nil)
	result := svc.ExecuteRebalanceOrders(recs, false, nil)

	if len(client.calls) != 2 {
		t.Errorf("want 2 API calls, got %d", len(client.calls))
	}
	if len(result.Executions) != 2 {
		t.Fatalf("want 2 executions, got %d", len(result.Executions))
	}
	if result.Executions[0].Status != "success" {
		t.Errorf("exec 0: want success, got %s", result.Executions[0].Status)
	}
	if result.Executions[1].Status != "failed" {
		t.Errorf("exec 1: want failed, got %s", result.Executions[1].Status)
	}
	if result.Executions[1].Message != "주문가능수량 부족" {
		t.Errorf("exec 1 message: want 주문가능수량 부족, got %s", result.Executions[1].Message)
	}
	if len(result.Deferred) != 1 || result.Deferred[0].Ticker != "005930" {
		t.Fatalf("deferred buys = %+v, want 005930", result.Deferred)
	}
}

func TestExecuteResultsPersisted(t *testing.T) {
	recs := []models.RebalanceRecommendation{
		makeRec("AAPL", models.ActionSell, "USD", "2"),
		makeRec("005930", models.ActionBuy, "KRW", "7"),
		makeRec("TSLA", models.ActionBuy, "USD", "0.3"), // skipped
	}

	okResp := map[string]any{"rt_cd": "0", "msg1": "완료"}
	client := &mockOrderClient{response: okResp}
	repo := &mockExecRepo{}

	svc := services.NewRebalanceExecutionService(client, repo, nil)
	svc.ExecuteRebalanceOrders(recs, false, nil)

	if len(repo.calls) != 3 { // 1 executed + 1 skipped + 1 deferred
		t.Errorf("want 3 repo.Create calls, got %d", len(repo.calls))
	}
	statuses := map[string]bool{}
	for _, c := range repo.calls {
		statuses[c.status] = true
	}
	if !statuses["success"] {
		t.Error("expected success status in persisted records")
	}
	if !statuses["skipped"] {
		t.Error("expected skipped status in persisted records")
	}
	if !statuses["deferred"] {
		t.Error("expected deferred status in persisted records")
	}
}

func TestExecuteSyncCalledAfter(t *testing.T) {
	recs := []models.RebalanceRecommendation{
		makeRec("005930", models.ActionBuy, "KRW", "7"),
	}
	client := &mockOrderClient{response: map[string]any{"rt_cd": "0", "msg1": ""}}
	sync := &mockSyncService{}

	svc := services.NewRebalanceExecutionService(client, nil, sync)
	svc.ExecuteRebalanceOrders(recs, false, nil)

	if sync.calls != 1 {
		t.Errorf("sync.SyncAccount() want 1 call, got %d", sync.calls)
	}
}

func TestExecuteSyncFailurePreservesResults(t *testing.T) {
	recs := []models.RebalanceRecommendation{
		makeRec("005930", models.ActionBuy, "KRW", "7"),
	}
	client := &mockOrderClient{response: map[string]any{"rt_cd": "0", "msg1": ""}}
	sync := &mockSyncService{err: errors.New("sync failed")}

	svc := services.NewRebalanceExecutionService(client, nil, sync)
	result := svc.ExecuteRebalanceOrders(recs, false, nil)

	if len(result.Executions) != 1 || result.Executions[0].Status != "success" {
		t.Error("executions should be preserved despite sync failure")
	}
	if result.SyncWarning == "" || result.SyncWarning != "sync failed" {
		t.Errorf("sync warning: want 'sync failed', got %q", result.SyncWarning)
	}
}

// satisfy numeric import
var _ = numeric.Zero
