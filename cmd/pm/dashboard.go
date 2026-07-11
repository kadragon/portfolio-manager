package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"sort"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/services"
)

// sortableDashboardKeys are the -sort values accepted by runDashboard, mapped
// to how each row's sort key is read off a GroupHoldingPair.
var sortableDashboardKeys = map[string]bool{"value": true, "1d": true, "1m": true, "6m": true, "1y": true}

// runDashboard prints the portfolio summary, falling back to a per-group
// holdings breakdown when no price service is configured or the summary
// computation fails for a price-service reason — mirrors DashboardHandler.index.
func runDashboard(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm dashboard", flag.ExitOnError)
	noChangeRates := fs.Bool("no-change-rates", false, "skip benchmark change-rate computation (faster)")
	sortKey := fs.String("sort", "", "sort holdings by {value,1d,1m,6m,1y}, best first (default: unsorted)")
	asc := fs.Bool("asc", false, "with -sort, order ascending (worst first) instead of descending")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if *sortKey != "" && !sortableDashboardKeys[*sortKey] {
		return fmt.Errorf("invalid -sort %q: must be one of value, 1d, 1m, 6m, 1y", *sortKey)
	}
	if *sortKey != "" && *sortKey != "value" && *noChangeRates {
		return fmt.Errorf("-sort %q requires change rates: drop -no-change-rates", *sortKey)
	}
	if *asc && *sortKey == "" {
		return fmt.Errorf("-asc requires -sort to be specified")
	}

	if c.Portfolio == nil {
		return fmt.Errorf("portfolio service not configured")
	}

	if c.Portfolio.HasPriceService() {
		summary, err := c.Portfolio.GetPortfolioSummary(ctx, !*noChangeRates)
		if err == nil {
			if *sortKey != "" {
				sortDashboardHoldings(summary.Holdings, *sortKey, *asc)
			}
			return printJSON(summary)
		}
		if !errors.Is(err, services.ErrNoPriceService) {
			return fmt.Errorf("portfolio summary: %w", err)
		}
	}

	// The per-group fallback below carries no ValueKRW/ChangeRates fields, so
	// -sort has nothing to operate on — fail loudly instead of silently ignoring it.
	if *sortKey != "" {
		return fmt.Errorf("-sort %q requires priced dashboard data (no price service configured)", *sortKey)
	}

	groupHoldings, err := c.Portfolio.GetHoldingsByGroup(ctx)
	if err != nil {
		return fmt.Errorf("holdings by group: %w", err)
	}
	return printJSON(groupHoldings)
}

// sortDashboardHoldings orders holdings in place by sortKey, best-first unless
// asc is set. Rows missing the requested value (nil ValueKRW, absent change-rate
// period) sort last regardless of direction. Note: GetStockChangeRates reports a
// literal 0 (not a missing key) when a stock has no cached price far enough back
// for the period — e.g. a ticker held less than a year shows "1y": 0 — so those
// rows are indistinguishable here from a genuine flat return and will not sort last.
func sortDashboardHoldings(holdings []models.GroupHoldingPair, sortKey string, asc bool) {
	keyOf := func(h models.GroupHoldingPair) (numeric.Decimal, bool) {
		if sortKey == "value" {
			if h.Holding.ValueKRW == nil {
				return numeric.Zero, false
			}
			return *h.Holding.ValueKRW, true
		}
		rate, ok := h.Holding.ChangeRates[sortKey]
		return rate, ok
	}

	sort.SliceStable(holdings, func(i, j int) bool {
		vi, oki := keyOf(holdings[i])
		vj, okj := keyOf(holdings[j])
		if oki != okj {
			return oki // present values sort before missing ones
		}
		if !oki {
			return false
		}
		if asc {
			return vi.Cmp(vj.Decimal) < 0
		}
		return vi.Cmp(vj.Decimal) > 0
	})
}
