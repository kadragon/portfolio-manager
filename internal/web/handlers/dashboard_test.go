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

func setupDashboard(t *testing.T) (*echo.Echo, *container.Container) {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	e := echo.New()
	handlers.NewDashboardHandler(c).Register(e)
	return e, c
}

func TestDashboardEmpty(t *testing.T) {
	e, _ := setupDashboard(t)
	rec := do(e, http.MethodGet, "/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	// With PriceService wired, even an empty DB renders the summary template;
	// the holdings table shows the "no holdings" row.
	if !strings.Contains(body, "보유 종목이 없습니다") {
		t.Error("empty holdings message missing")
	}
	if !strings.Contains(body, "대시보드") {
		t.Error("page title missing")
	}
}

func TestDashboardNilPortfolio(t *testing.T) {
	e, c := setupDashboard(t)
	c.Portfolio = nil
	rec := do(e, http.MethodGet, "/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestDashboardNoPriceService(t *testing.T) {
	e, c := setupDashboard(t)
	// Override with nil priceService → HasPriceService() returns false → fallback to GroupHoldings
	c.Portfolio = services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, nil, nil)
	rec := do(e, http.MethodGet, "/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestDashboardFallback(t *testing.T) {
	e, c := setupDashboard(t)
	ctx := context.Background()

	group, err := c.Groups.Create(ctx, "테스트그룹", 100.0)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	stock, err := c.Stocks.Create(ctx, "005930", group.ID)
	if err != nil {
		t.Fatalf("create stock: %v", err)
	}
	account, err := c.Accounts.Create(ctx, "테스트계좌", numeric.Zero)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	qty, _ := numeric.FromString("100")
	_, err = c.Holdings.Create(ctx, account.ID, stock.ID, qty)
	if err != nil {
		t.Fatalf("create holding: %v", err)
	}

	rec := do(e, http.MethodGet, "/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "005930") {
		t.Error("ticker missing from fallback table")
	}
	if !strings.Contains(body, "테스트그룹") {
		t.Error("group name missing")
	}
}
