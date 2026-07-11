package main

import (
	"context"
	"fmt"

	"github.com/kadragon/portfolio-manager/internal/container"
)

// runPriceSync replaces the removed 15-minute background ticker: it runs one
// price-sync pass on demand. SyncOnce logs per-ticker failures internally and
// never returns an error, so this always reports success if it ran at all.
func runPriceSync(ctx context.Context, c *container.Container, _ []string) error {
	if c.PriceSync == nil {
		return fmt.Errorf("KIS price sync service not configured (.env KIS_APP_KEY/KIS_APP_SECRET)")
	}
	c.PriceSync.SyncOnce(ctx)
	return printJSON(map[string]string{"status": "synced"})
}
