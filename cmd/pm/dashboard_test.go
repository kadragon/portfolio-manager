package main

import (
	"context"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/services"
)

func pairWithChangeRate(t *testing.T, ticker, oneMonth string) models.GroupHoldingPair {
	t.Helper()
	return models.GroupHoldingPair{
		Holding: models.StockHoldingWithPrice{
			Stock:       models.Stock{Ticker: ticker},
			ChangeRates: map[string]numeric.Decimal{"1m": mustDecimal(t, oneMonth)},
		},
	}
}

func TestSortDashboardHoldingsByChangeRateDescending(t *testing.T) {
	holdings := []models.GroupHoldingPair{
		pairWithChangeRate(t, "WORST", "-2.2"),
		pairWithChangeRate(t, "BEST", "3.8"),
		pairWithChangeRate(t, "MID", "1.0"),
	}

	sortDashboardHoldings(holdings, "1m", false)

	got := []string{holdings[0].Holding.Stock.Ticker, holdings[1].Holding.Stock.Ticker, holdings[2].Holding.Stock.Ticker}
	want := []string{"BEST", "MID", "WORST"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestSortDashboardHoldingsAscending(t *testing.T) {
	holdings := []models.GroupHoldingPair{
		pairWithChangeRate(t, "BEST", "3.8"),
		pairWithChangeRate(t, "WORST", "-2.2"),
	}

	sortDashboardHoldings(holdings, "1m", true)

	if holdings[0].Holding.Stock.Ticker != "WORST" {
		t.Fatalf("expected WORST first ascending, got %s", holdings[0].Holding.Stock.Ticker)
	}
}

func TestSortDashboardHoldingsMissingKeySortsLast(t *testing.T) {
	missing := models.GroupHoldingPair{
		Holding: models.StockHoldingWithPrice{
			Stock:       models.Stock{Ticker: "NODATA"},
			ChangeRates: map[string]numeric.Decimal{},
		},
	}
	holdings := []models.GroupHoldingPair{missing, pairWithChangeRate(t, "HASDATA", "1.0")}

	sortDashboardHoldings(holdings, "1m", false)

	if holdings[0].Holding.Stock.Ticker != "HASDATA" {
		t.Fatalf("expected row with data first, got %s", holdings[0].Holding.Stock.Ticker)
	}
	if holdings[1].Holding.Stock.Ticker != "NODATA" {
		t.Fatalf("expected row missing 1m to sort last, got %s", holdings[1].Holding.Stock.Ticker)
	}
}

func TestRunDashboardRejectsAscWithoutSort(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)

	if err := runDashboard(ctx, c, []string{"-asc"}); err == nil {
		t.Fatal("expected error when -asc is passed without -sort")
	}
}

func TestRunDashboardRejectsSortWithoutPriceService(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	// container.New/NewWithQueries always wire a DB-backed PriceService regardless
	// of KIS config, so simulate the no-price-service fallback (GetHoldingsByGroup
	// path) directly, matching how PortfolioService documents priceService as optional.
	c.Portfolio = services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, nil, nil)

	if c.Portfolio.HasPriceService() {
		t.Fatal("expected no price service wired")
	}
	if err := runDashboard(ctx, c, []string{"-sort", "value"}); err == nil {
		t.Fatal("expected error when -sort is used without a price service")
	}
}
