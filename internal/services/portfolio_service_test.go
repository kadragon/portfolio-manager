package services_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/services"
)

func newPortfolioContainer(t *testing.T) *container.Container {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return container.NewWithQueries(sqlDB, q)
}

func TestHasPriceServiceFalse(t *testing.T) {
	c := newPortfolioContainer(t)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, nil, nil)
	if ps.HasPriceService() {
		t.Error("HasPriceService() with nil priceService should be false")
	}
}

func TestHasPriceServiceTrue(t *testing.T) {
	c := newPortfolioContainer(t)
	priceService := services.NewPriceService(c.StockPrices)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)
	if !ps.HasPriceService() {
		t.Error("HasPriceService() with non-nil priceService should be true")
	}
}

func TestGetHoldingsByGroupEmpty(t *testing.T) {
	c := newPortfolioContainer(t)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, nil, nil)
	groups, err := ps.GetHoldingsByGroup(context.Background())
	if err != nil {
		t.Fatalf("GetHoldingsByGroup: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected empty, got %d groups", len(groups))
	}
}

func TestGetHoldingsByGroupWithData(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "성장주", 100.0)
	s, _ := c.Stocks.Create(ctx, "005930", g.ID)
	acc, _ := c.Accounts.Create(ctx, "내 계좌", numeric.Zero)
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(10))

	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, nil, nil)
	groups, err := ps.GetHoldingsByGroup(ctx)
	if err != nil {
		t.Fatalf("GetHoldingsByGroup: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Group.Name != "성장주" {
		t.Errorf("group name = %q, want 성장주", groups[0].Group.Name)
	}
}

func TestGetPortfolioSummaryNoPriceService(t *testing.T) {
	c := newPortfolioContainer(t)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, nil, nil)
	_, err := ps.GetPortfolioSummary(context.Background(), false)
	if !errors.Is(err, services.ErrNoPriceService) {
		t.Errorf("expected ErrNoPriceService, got %v", err)
	}
}

func TestGetPortfolioSummaryWithDBPrices(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	priceService := services.NewPriceService(c.StockPrices)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)

	summary, err := ps.GetPortfolioSummary(ctx, false)
	if err != nil {
		t.Fatalf("GetPortfolioSummary: %v", err)
	}
	if summary == nil {
		t.Fatal("summary is nil")
	}
}

func TestGetPortfolioSummaryWithData(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "성장주", 50.0)
	s, _ := c.Stocks.Create(ctx, "005930", g.ID)
	acc, _ := c.Accounts.Create(ctx, "내 계좌", numeric.FromInt(1000000))
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(10))

	// Seed price into DB (PriceService is DB-only).
	today, _ := datex.ParseDate("2026-06-01")
	p, _ := numeric.FromString("74000")
	_, _ = c.StockPrices.Save(ctx, "005930", today, p, "KRW", "삼성전자", sql.NullString{})

	priceService := services.NewPriceService(c.StockPrices)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)

	summary, err := ps.GetPortfolioSummary(ctx, false)
	if err != nil {
		t.Fatalf("GetPortfolioSummary with data: %v", err)
	}
	if summary == nil {
		t.Fatal("summary is nil")
	}
	groupRows := services.ComputeGroupSummary(summary)
	if len(groupRows) == 0 {
		t.Error("ComputeGroupSummary returned empty rows for non-empty portfolio")
	}
}

func TestGetPortfolioSummaryUSDStock(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "해외주", 50.0)
	exchange := "NASD"
	s, _ := c.Stocks.Create(ctx, "AAPL", g.ID)
	_, _ = c.Stocks.UpdateExchange(ctx, s.ID, exchange)
	s.Exchange = &exchange
	acc, _ := c.Accounts.Create(ctx, "해외계좌", numeric.Zero)
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(5))

	// Seed USD price into DB.
	today, _ := datex.ParseDate("2026-06-01")
	p, _ := numeric.FromString("195.89")
	_, _ = c.StockPrices.Save(ctx, "AAPL", today, p, "USD", "Apple Inc.", sql.NullString{String: "NASD", Valid: true})

	rate, _ := numeric.FromString("1300")
	exchangeRate := services.NewFixedExchangeRateService(rate)
	priceService := services.NewPriceService(c.StockPrices)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, exchangeRate)

	summary, err := ps.GetPortfolioSummary(ctx, false)
	if err != nil {
		t.Fatalf("GetPortfolioSummary USD: %v", err)
	}
	if summary == nil {
		t.Fatal("summary is nil")
	}
}

func TestResolveAndPersistNameWithPriceService(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "테스트그룹", 100.0)
	stock, _ := c.Stocks.Create(ctx, "AAPL", g.ID)

	priceService := services.NewPriceService(c.StockPrices)
	ss := services.NewStockService(c.Stocks, priceService)
	result := ss.ResolveAndPersistName(ctx, &stock)
	// DB has no price/name data, result is ""
	_ = result
}
