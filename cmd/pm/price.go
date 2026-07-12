package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/numeric"
)

func runPrice(ctx context.Context, c *container.Container, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pm price list|set|delete [flags]")
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "list":
		return priceList(ctx, c, rest)
	case "set":
		return priceSet(ctx, c, rest)
	case "delete":
		return priceDelete(ctx, c, rest)
	default:
		return fmt.Errorf("unknown price verb %q", verb)
	}
}

func priceSet(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm price set", flag.ExitOnError)
	tickerRaw := fs.String("ticker", "", "stock ticker (required)")
	dateRaw := fs.String("date", "", "price date YYYY-MM-DD (required)")
	priceRaw := fs.String("price", "", "closing price (required)")
	currencyRaw := fs.String("currency", "", "currency code (required for a new row)")
	nameRaw := fs.String("name", "", "stock name (required for a new row)")
	exchangeRaw := fs.String("exchange", "", `exchange code; "/clear" unsets it`)
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	seen := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { seen[f.Name] = true })

	ticker := strings.ToUpper(strings.TrimSpace(*tickerRaw))
	if ticker == "" {
		return fmt.Errorf("-ticker is required")
	}
	priceDate, err := datex.ParseDate(strings.TrimSpace(*dateRaw))
	if err != nil {
		return fmt.Errorf("invalid -date: %w", err)
	}
	price, err := numeric.FromString(strings.TrimSpace(*priceRaw))
	if err != nil {
		return fmt.Errorf("invalid -price: %w", err)
	}
	if !price.IsPositive() {
		return fmt.Errorf("-price must be positive")
	}
	existing, err := c.StockPrices.GetByTickerAndDate(ctx, ticker, priceDate)
	if err != nil {
		return fmt.Errorf("get cached stock price: %w", err)
	}

	currency := ""
	name := ""
	exchange := sql.NullString{}
	if existing != nil {
		currency = existing.Currency
		name = existing.Name
		exchange = existing.Exchange
	}
	if seen["currency"] {
		currency = strings.ToUpper(strings.TrimSpace(*currencyRaw))
	}
	if seen["name"] {
		name = strings.TrimSpace(*nameRaw)
	}
	if seen["exchange"] {
		value := strings.TrimSpace(*exchangeRaw)
		if value == "" || strings.EqualFold(value, "/clear") {
			exchange = sql.NullString{}
		} else {
			exchange = sql.NullString{String: strings.ToUpper(value), Valid: true}
		}
	}
	if existing == nil {
		if currency == "" {
			return fmt.Errorf("-currency is required for a new cached price")
		}
		if name == "" {
			return fmt.Errorf("-name is required for a new cached price")
		}
	}

	stored, err := c.StockPrices.Save(ctx, ticker, priceDate, price, currency, name, exchange)
	if err != nil {
		return fmt.Errorf("set cached stock price: %w", err)
	}
	return printJSON(stored)
}

func priceDelete(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm price delete", flag.ExitOnError)
	tickerRaw := fs.String("ticker", "", "stock ticker (required)")
	dateRaw := fs.String("date", "", "price date YYYY-MM-DD (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	ticker := strings.ToUpper(strings.TrimSpace(*tickerRaw))
	if ticker == "" {
		return fmt.Errorf("-ticker is required")
	}
	priceDate, err := datex.ParseDate(strings.TrimSpace(*dateRaw))
	if err != nil {
		return fmt.Errorf("invalid -date: %w", err)
	}
	deleted, err := c.StockPrices.Delete(ctx, ticker, priceDate)
	if err != nil {
		return fmt.Errorf("delete cached stock price: %w", err)
	}
	if !deleted {
		return fmt.Errorf("cached stock price %s on %s not found", ticker, priceDate.ISO())
	}
	return printJSON(map[string]string{"status": "deleted", "ticker": ticker, "date": priceDate.ISO()})
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
