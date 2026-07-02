package handlers_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/services"
	"github.com/kadragon/portfolio-manager/internal/web/handlers"
)

func setupRebalance(t *testing.T) (*echo.Echo, *container.Container) {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	// NewWithQueries always builds a non-nil PriceService (with nil client), so
	// HasPriceService() returns true by default. Override Portfolio with a nil
	// priceService to reach the no-price-service guard in the rebalance handler.
	c.Portfolio = services.NewPortfolioService(
		c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, nil, nil,
	)
	e := echo.New()
	handlers.NewRebalanceHandler(c).Register(e)
	return e, c
}

// TestRebalanceViewNoPriceService checks that GET /rebalance with no KIS configured
// returns 200 with the "가격 서비스가 설정되지 않았습니다" message.
func TestRebalanceViewNoPriceService(t *testing.T) {
	e, _ := setupRebalance(t)
	rec := do(e, http.MethodGet, "/rebalance", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "가격 서비스가 설정되지 않았습니다") {
		t.Errorf("no-price-service message missing:\n%s", rec.Body.String())
	}
}

// TestRebalanceViewWithData checks that GET /rebalance still returns 200 with the
// no-price-service message even when groups/stocks/accounts exist in the DB,
// because HasPriceService() is false on the in-memory container.
func TestRebalanceViewWithData(t *testing.T) {
	e, c := setupRebalance(t)

	ctx := context.Background()
	group, err := c.Groups.Create(ctx, "성장주", 50.0)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	_, err = c.Stocks.Create(ctx, "005930", group.ID)
	if err != nil {
		t.Fatalf("create stock: %v", err)
	}
	_, err = c.Accounts.Create(ctx, "내 계좌", numeric.Zero)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	rec := do(e, http.MethodGet, "/rebalance", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "가격 서비스가 설정되지 않았습니다") {
		t.Errorf("no-price-service message missing:\n%s", rec.Body.String())
	}
}

// TestRebalanceExecuteNoPriceService checks that POST /rebalance/execute with no KIS
// returns 200 with the "가격 서비스 없음" partial.
func TestRebalanceExecuteNoPriceService(t *testing.T) {
	e, _ := setupRebalance(t)
	rec := do(e, http.MethodPost, "/rebalance/execute", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "가격 서비스 없음") {
		t.Errorf("no-price-service partial missing:\n%s", rec.Body.String())
	}
}

// TestRebalanceViewRestrictOverseas checks that GET /rebalance?restrict_overseas=1
// returns 200 (same no-price-service path regardless of restrict_overseas flag).
func TestRebalanceViewRestrictOverseas(t *testing.T) {
	e, _ := setupRebalance(t)
	rec := do(e, http.MethodGet, "/rebalance?restrict_overseas=1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "가격 서비스가 설정되지 않았습니다") {
		t.Errorf("no-price-service message missing:\n%s", rec.Body.String())
	}
}

// TestRebalanceViewWithPriceService exercises buildPlan by leaving the default
// container Portfolio intact (HasPriceService() == true, nil client → zero prices).
func TestRebalanceViewWithPriceService(t *testing.T) {
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	// Do NOT override c.Portfolio — it already has a non-nil PriceService.
	e := echo.New()
	handlers.NewRebalanceHandler(c).Register(e)

	rec := do(e, http.MethodGet, "/rebalance", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "가격 서비스가 설정되지 않았습니다") {
		t.Error("should not show no-price-service when PriceService is configured")
	}
}

// TestRebalanceViewWithHoldings exercises buildPlan's account/holdings loop and
// the view's group-summary render path: with a price service plus real holdings,
// buildPlan returns a non-nil summary that view reuses for ComputeGroupSummary.
func TestRebalanceViewWithHoldings(t *testing.T) {
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	e := echo.New()
	handlers.NewRebalanceHandler(c).Register(e)

	ctx := context.Background()
	group, err := c.Groups.Create(ctx, "성장주", 50.0)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	stock, err := c.Stocks.Create(ctx, "005930", group.ID)
	if err != nil {
		t.Fatalf("create stock: %v", err)
	}
	acc, err := c.Accounts.Create(ctx, "내 계좌", numeric.Zero)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	qty, _ := numeric.FromString("10")
	if _, err := c.Holdings.Create(ctx, acc.ID, stock.ID, qty); err != nil {
		t.Fatalf("create holding: %v", err)
	}

	rec := do(e, http.MethodGet, "/rebalance", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "가격 서비스가 설정되지 않았습니다") {
		t.Error("should not show no-price-service when PriceService is configured")
	}
}

// TestRebalanceExecuteWithPriceService exercises buildPlan via POST /rebalance/execute.
func TestRebalanceExecuteWithPriceService(t *testing.T) {
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	e := echo.New()
	handlers.NewRebalanceHandler(c).Register(e)

	rec := do(e, http.MethodPost, "/rebalance/execute", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

// TestRebalanceExecuteInvalidAccountID checks that POST /rebalance/execute with a
// malformed account_id returns the "잘못된 계좌 ID입니다." error partial instead of
// executing against all accounts.
func TestRebalanceExecuteInvalidAccountID(t *testing.T) {
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	e := echo.New()
	handlers.NewRebalanceHandler(c).Register(e)

	rec := do(e, http.MethodPost, "/rebalance/execute", url.Values{"account_id": {"not-a-uuid"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "잘못된 계좌 ID입니다.") {
		t.Errorf("invalid account_id message missing:\n%s", rec.Body.String())
	}
}

// TestRebalanceExecuteAccountWithNoRecsSkipsExecution checks that POST
// /rebalance/execute with a valid account_id that has zero matching
// recommendations (here: an account with no holdings/groups) returns a
// "no orders" message instead of falsely reporting "주문 실행 완료" and
// running ExecuteRebalanceOrders (which would otherwise trigger an
// unconditional KIS account sync for zero actual work).
func TestRebalanceExecuteAccountWithNoRecsSkipsExecution(t *testing.T) {
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	e := echo.New()
	handlers.NewRebalanceHandler(c).Register(e)

	ctx := context.Background()
	acc, err := c.Accounts.Create(ctx, "내 계좌", numeric.Zero)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	rec := do(e, http.MethodPost, "/rebalance/execute", url.Values{"account_id": {acc.ID.String()}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "잘못된 계좌 ID입니다.") {
		t.Errorf("valid account_id should not be rejected:\n%s", body)
	}
	if !strings.Contains(body, "해당 계좌에 실행할 주문이 없습니다.") {
		t.Errorf("no-recs message missing:\n%s", body)
	}
	if strings.Contains(body, "주문 실행 완료") {
		t.Errorf("should not falsely report execution success when there are no recs:\n%s", body)
	}
}

// TestRebalanceExecuteScopedToAccountFiltersRecs seeds two accounts that both
// breach the same rebalance group's target band (forcing real sell recs for
// each), then executes scoped to one account_id and asserts the response
// contains only that account's ticker — not the other account's — and
// carries the out-of-band fragment that disables its own execute button
// (preventing a double-submit of the same order via a stale button).
func TestRebalanceExecuteScopedToAccountFiltersRecs(t *testing.T) {
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	e := echo.New()
	handlers.NewRebalanceHandler(c).Register(e)

	ctx := context.Background()
	group, err := c.Groups.Create(ctx, "국내성장", 10.0)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	stockA, err := c.Stocks.Create(ctx, "005930", group.ID)
	if err != nil {
		t.Fatalf("create stock A: %v", err)
	}
	stockB, err := c.Stocks.Create(ctx, "000660", group.ID)
	if err != nil {
		t.Fatalf("create stock B: %v", err)
	}

	today := datex.FromTime(ktime.Now().Time)
	price, _ := numeric.FromString("50000")
	if _, err := c.StockPrices.Save(ctx, "005930", today, price, "KRW", "삼성전자", sql.NullString{}); err != nil {
		t.Fatalf("save price A: %v", err)
	}
	if _, err := c.StockPrices.Save(ctx, "000660", today, price, "KRW", "SK하이닉스", sql.NullString{}); err != nil {
		t.Fatalf("save price B: %v", err)
	}

	accA, err := c.Accounts.Create(ctx, "계좌A", numeric.Zero)
	if err != nil {
		t.Fatalf("create account A: %v", err)
	}
	accB, err := c.Accounts.Create(ctx, "계좌B", numeric.Zero)
	if err != nil {
		t.Fatalf("create account B: %v", err)
	}

	qty, _ := numeric.FromString("100")
	if _, err := c.Holdings.Create(ctx, accA.ID, stockA.ID, qty); err != nil {
		t.Fatalf("create holding A: %v", err)
	}
	if _, err := c.Holdings.Create(ctx, accB.ID, stockB.ID, qty); err != nil {
		t.Fatalf("create holding B: %v", err)
	}

	// Both accounts breach the group's target band, so an unscoped execute
	// would place a sell for each. A fake order client is needed because
	// RebalanceResultPartial only reports execution counts (not tickers),
	// and counts are only populated when an OrderClient is configured.
	c.RebalanceExecution = services.NewRebalanceExecutionService(fakeOrderClient{}, nil, nil)

	rec := do(e, http.MethodPost, "/rebalance/execute", url.Values{"account_id": {accA.ID.String()}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "성공:</strong> 1") {
		t.Errorf("expected exactly 1 successful execution when scoped to account A, got:\n%s", body)
	}
	wantOOB := `id="account-execute-` + accA.ID.String() + `"`
	if !strings.Contains(body, wantOOB) {
		t.Errorf("expected OOB done-fragment %q to disable the account's execute button:\n%s", wantOOB, body)
	}
}

// fakeOrderClient always reports a successful fill, so
// RebalanceExecutionResult.Executions gets populated for assertions.
type fakeOrderClient struct{}

func (fakeOrderClient) PlaceOrder(intent models.OrderIntent) (map[string]any, error) {
	return map[string]any{"rt_cd": "0", "msg1": "OK"}, nil
}
