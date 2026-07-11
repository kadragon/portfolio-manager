package main

import (
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
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
