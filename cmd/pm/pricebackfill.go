package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/datex"
)

// runPriceBackfill fetches every daily close KIS has for one ticker across an
// arbitrary [from, to] date range and stores whatever isn't already cached —
// for on-demand history gaps SyncOnce's fixed 1y/6m/1m/1d checkpoints don't cover.
func runPriceBackfill(ctx context.Context, c *container.Container, args []string) error {
	if c.PriceSync == nil {
		return fmt.Errorf("KIS price sync service not configured (.env KIS_APP_KEY/KIS_APP_SECRET)")
	}

	fs := flag.NewFlagSet("price-backfill", flag.ContinueOnError)
	ticker := fs.String("ticker", "", "stock ticker (required)")
	from := fs.String("from", "", "start date YYYY-MM-DD (required)")
	to := fs.String("to", "", "end date YYYY-MM-DD (required)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if *ticker == "" || *from == "" || *to == "" {
		return fmt.Errorf("price-backfill: -ticker, -from, -to are required")
	}
	start, err := datex.ParseDate(*from)
	if err != nil {
		return fmt.Errorf("invalid -from: %w", err)
	}
	end, err := datex.ParseDate(*to)
	if err != nil {
		return fmt.Errorf("invalid -to: %w", err)
	}

	result, err := c.PriceSync.BackfillRange(ctx, *ticker, start, end)
	if err != nil {
		return fmt.Errorf("price backfill: %w", err)
	}
	return printJSON(result)
}
