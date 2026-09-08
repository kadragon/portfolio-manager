package main

import (
	"context"
	"fmt"

	"github.com/kadragon/portfolio-manager/internal/container"
)

// runPriceSync replaces the removed 15-minute background ticker: it runs one
// price-sync pass on demand. SyncOnce logs per-ticker failures internally and
// never returns an error, so the pass itself always "succeeds"; the reported
// status distinguishes a complete run from one that left benchmark deposit dates
// unpriced, which silently blanks the timing-matched dashboard comparison.
func runPriceSync(ctx context.Context, c *container.Container, _ []string) error {
	if c.PriceSync == nil {
		return fmt.Errorf("KIS price sync service not configured (.env KIS_APP_KEY/KIS_APP_SECRET)")
	}
	result := c.PriceSync.SyncOnce(ctx)
	status := "synced"
	if result.UnpricedBenchmarkDates > 0 {
		status = "synced_incomplete"
	}
	return printJSON(map[string]any{
		"status":                   status,
		"unpriced_benchmark_dates": result.UnpricedBenchmarkDates,
	})
}
