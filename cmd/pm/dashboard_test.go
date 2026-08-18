package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/kadragon/portfolio-manager/internal/datex"
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

func TestRunDashboardSortsMissingOneYearHistoryLast(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	fixedToday := datex.New(2026, time.June, 1).Time

	group, err := c.Groups.Create(ctx, "Dashboard test", 100)
	if err != nil {
		t.Fatalf("Groups.Create: %v", err)
	}
	longHistory, err := c.Stocks.Create(ctx, "LONG", group.ID)
	if err != nil {
		t.Fatalf("Stocks.Create LONG: %v", err)
	}
	shortHistory, err := c.Stocks.Create(ctx, "SHORT", group.ID)
	if err != nil {
		t.Fatalf("Stocks.Create SHORT: %v", err)
	}
	account, err := c.Accounts.Create(ctx, "Dashboard test account", numeric.Zero)
	if err != nil {
		t.Fatalf("Accounts.Create: %v", err)
	}
	for _, stock := range []models.Stock{longHistory, shortHistory} {
		if _, err := c.Holdings.Create(ctx, account.ID, stock.ID, numeric.FromInt(1)); err != nil {
			t.Fatalf("Holdings.Create %s: %v", stock.Ticker, err)
		}
	}

	savePrice := func(ticker, isoDate, price, name string) {
		t.Helper()
		priceDate, err := datex.ParseDate(isoDate)
		if err != nil {
			t.Fatalf("ParseDate %s: %v", isoDate, err)
		}
		if _, err := c.StockPrices.Save(ctx, ticker, priceDate, mustDecimal(t, price), "KRW", name, sql.NullString{}); err != nil {
			t.Fatalf("StockPrices.Save %s %s: %v", ticker, isoDate, err)
		}
	}

	// LONG has a close on the computed 1y target (2025-05-30). SHORT has
	// current and recent history only, so its 1y key must be absent.
	savePrice("LONG", "2026-06-01", "100", "Long history")
	savePrice("LONG", "2025-05-30", "80", "Long history")
	savePrice("SHORT", "2026-06-01", "200", "Short history")
	savePrice("SHORT", "2026-05-01", "190", "Short history")

	priceService := services.NewPriceService(c.StockPrices).WithTodayProvider(func() time.Time {
		return fixedToday
	})
	c.Portfolio = services.NewPortfolioService(
		c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil,
	)

	data := captureDashboardOutput(t, func() error {
		return runDashboard(ctx, c, []string{"-sort", "1y"})
	})
	var dashboard struct {
		Holdings []struct {
			Holding struct {
				Stock struct {
					Ticker string `json:"Ticker"`
				} `json:"Stock"`
				ChangeRates map[string]json.RawMessage `json:"ChangeRates"`
			} `json:"Holding"`
		} `json:"Holdings"`
	}
	if err := json.Unmarshal(data, &dashboard); err != nil {
		t.Fatalf("decode dashboard output: %v\n%s", err, data)
	}
	if len(dashboard.Holdings) != 2 {
		t.Fatalf("dashboard holdings = %d, want 2", len(dashboard.Holdings))
	}
	gotOrder := []string{
		dashboard.Holdings[0].Holding.Stock.Ticker,
		dashboard.Holdings[1].Holding.Stock.Ticker,
	}
	if want := []string{"LONG", "SHORT"}; gotOrder[0] != want[0] || gotOrder[1] != want[1] {
		t.Fatalf("dashboard -sort 1y order = %v, want %v", gotOrder, want)
	}
	if _, ok := dashboard.Holdings[0].Holding.ChangeRates["1y"]; !ok {
		t.Fatal("LONG dashboard row missing 1y change-rate key")
	}
	if _, ok := dashboard.Holdings[1].Holding.ChangeRates["1y"]; ok {
		t.Fatal("SHORT dashboard row unexpectedly contains 1y change-rate key")
	}
}

func captureDashboardOutput(t *testing.T, fn func() error) []byte {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = original }()
	callErr := fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("close dashboard output: %v", err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	if callErr != nil {
		t.Fatalf("runDashboard: %v", callErr)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read dashboard output: %v", err)
	}
	return data
}

func TestRunDashboardRejectsInvalidBenchmarkMode(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)

	if err := runDashboard(ctx, c, []string{"-benchmark-mode", "dollar-cost"}); err == nil {
		t.Fatal("expected error for an unknown -benchmark-mode value")
	}
}

func TestRunDashboardRejectsBenchmarkModeWithoutPriceService(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	// Same fallback simulation as TestRunDashboardRejectsSortWithoutPriceService.
	c.Portfolio = services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, nil, nil)

	if err := runDashboard(ctx, c, []string{"-benchmark-mode", "timing-matched"}); err == nil {
		t.Fatal("expected error when -benchmark-mode timing-matched is used without a price service")
	}
}
