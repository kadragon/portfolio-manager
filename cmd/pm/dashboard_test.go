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
	// Peer must be negative: descending, a missing key read as the zero value would
	// outrank it, so a comparator that dropped the present-before-missing guard
	// fails here. A positive peer would pass either way and pin nothing.
	holdings := []models.GroupHoldingPair{missing, pairWithChangeRate(t, "HASDATA", "-4.5")}

	sortDashboardHoldings(holdings, "1m", false)

	if holdings[0].Holding.Stock.Ticker != "HASDATA" {
		t.Fatalf("expected row with data first, got %s", holdings[0].Holding.Stock.Ticker)
	}
	if holdings[1].Holding.Stock.Ticker != "NODATA" {
		t.Fatalf("expected row missing 1m to sort last, got %s", holdings[1].Holding.Stock.Ticker)
	}
}

// TestSortDashboardHoldingsMissingKeySortsLastAscending pins the direction-independent
// half of the rule: a row whose period key is absent (no history that far back) stays
// last ascending too. The peer must be positive — ascending, a missing key read as the
// zero value would sort ahead of it, so this fails without the present-before-missing
// guard, which a negative peer would not detect.
func TestSortDashboardHoldingsMissingKeySortsLastAscending(t *testing.T) {
	missing := models.GroupHoldingPair{
		Holding: models.StockHoldingWithPrice{
			Stock:       models.Stock{Ticker: "NODATA"},
			ChangeRates: map[string]numeric.Decimal{},
		},
	}
	holdings := []models.GroupHoldingPair{missing, pairWithChangeRate(t, "WINNER", "2.0")}

	sortDashboardHoldings(holdings, "1m", true)

	if holdings[0].Holding.Stock.Ticker != "WINNER" {
		t.Fatalf("expected row with data first ascending, got %s", holdings[0].Holding.Stock.Ticker)
	}
	if holdings[1].Holding.Stock.Ticker != "NODATA" {
		t.Fatalf("expected row missing 1m to sort last ascending, got %s", holdings[1].Holding.Stock.Ticker)
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
