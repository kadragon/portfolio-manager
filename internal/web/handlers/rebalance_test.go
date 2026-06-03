package handlers_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/db"
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
