package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/ktime"
)

func runPrice(ctx context.Context, c *container.Container, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pm price list [flags]")
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "list":
		return priceList(ctx, c, rest)
	default:
		return fmt.Errorf("unknown price verb %q", verb)
	}
}

// priceList reads already-cached daily closes for one ticker from stock_prices —
// a read-only counterpart to price-sync/price-backfill, for inspecting what's
// stored without reaching for sqlite3 directly.
func priceList(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm price list", flag.ExitOnError)
	ticker := fs.String("ticker", "", "stock ticker (required)")
	from := fs.String("from", "", "start date YYYY-MM-DD (default: earliest cached)")
	to := fs.String("to", "", "end date YYYY-MM-DD (default: today)")
	limit := fs.Int64("limit", 30, "max rows to return, newest first")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	tick := strings.ToUpper(strings.TrimSpace(*ticker))
	if tick == "" {
		return fmt.Errorf("-ticker is required")
	}
	if *limit <= 0 {
		return fmt.Errorf("-limit must be positive")
	}

	fromDate := datex.New(1900, 1, 1) // sentinel: no lower bound
	if strings.TrimSpace(*from) != "" {
		d, err := datex.ParseDate(*from)
		if err != nil {
			return fmt.Errorf("invalid -from: %w", err)
		}
		fromDate = d
	}
	toDate := datex.FromTime(ktime.Now().Time)
	if strings.TrimSpace(*to) != "" {
		d, err := datex.ParseDate(*to)
		if err != nil {
			return fmt.Errorf("invalid -to: %w", err)
		}
		toDate = d
	}

	if fromDate.Time.After(toDate.Time) {
		return fmt.Errorf("-from %s is after -to %s", fromDate.ISO(), toDate.ISO())
	}

	prices, err := c.StockPrices.ListByTickerRange(ctx, tick, fromDate, toDate, *limit)
	if err != nil {
		return fmt.Errorf("list stock prices: %w", err)
	}
	return printJSON(prices)
}
