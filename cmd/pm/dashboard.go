package main

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/services"
)

// runDashboard prints the portfolio summary, falling back to a per-group
// holdings breakdown when no price service is configured or the summary
// computation fails for a price-service reason — mirrors DashboardHandler.index.
func runDashboard(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm dashboard", flag.ExitOnError)
	noChangeRates := fs.Bool("no-change-rates", false, "skip benchmark change-rate computation (faster)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	if c.Portfolio == nil {
		return fmt.Errorf("portfolio service not configured")
	}

	if c.Portfolio.HasPriceService() {
		summary, err := c.Portfolio.GetPortfolioSummary(ctx, !*noChangeRates)
		if err == nil {
			return printJSON(summary)
		}
		if !errors.Is(err, services.ErrNoPriceService) {
			return fmt.Errorf("portfolio summary: %w", err)
		}
	}

	groupHoldings, err := c.Portfolio.GetHoldingsByGroup(ctx)
	if err != nil {
		return fmt.Errorf("holdings by group: %w", err)
	}
	return printJSON(groupHoldings)
}
