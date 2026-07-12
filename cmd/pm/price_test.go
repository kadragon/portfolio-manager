package main

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/numeric"
)

func TestPriceListRequiresTicker(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)

	if err := runPrice(ctx, c, []string{"list"}); err == nil {
		t.Fatal("expected error when -ticker is omitted")
	}
}

func TestPriceListReturnsCachedRows(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)

	for _, ds := range []string{"2026-01-10", "2026-01-15", "2026-01-20"} {
		d, _ := datex.ParseDate(ds)
		p, _ := numeric.FromString("100")
		if _, err := c.StockPrices.Save(ctx, "005930", d, p, "KRW", "삼성전자", sql.NullString{}); err != nil {
			t.Fatalf("seed price %s: %v", ds, err)
		}
	}

	if err := runPrice(ctx, c, []string{"list", "-ticker", "005930", "-limit", "2"}); err != nil {
		t.Fatalf("price list: %v", err)
	}

	from, _ := datex.ParseDate("1900-01-01")
	to, _ := datex.ParseDate("2026-12-31")
	got, err := c.StockPrices.ListByTickerRange(ctx, "005930", from, to, 2)
	if err != nil {
		t.Fatalf("list by range: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected limit=2 to cap result, got %d rows", len(got))
	}
	if got[0].PriceDate.Format("2006-01-02") != "2026-01-20" {
		t.Errorf("row 0 date = %v, want newest-first 2026-01-20", got[0].PriceDate)
	}
}

func TestPriceSetCreatesAndUpdatesCachedRow(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)

	if err := runPrice(ctx, c, []string{
		"set", "-ticker", "aapl", "-date", "2026-01-20", "-price", "100",
		"-currency", "USD", "-name", "Apple Inc.", "-exchange", "NASD",
	}); err != nil {
		t.Fatalf("price set create: %v", err)
	}
	d, _ := datex.ParseDate("2026-01-20")
	got, err := c.StockPrices.GetByTickerAndDate(ctx, "AAPL", d)
	if err != nil || got == nil {
		t.Fatalf("get created price: got=%v err=%v", got, err)
	}
	if got.Price.String() != "100" || got.Currency != "USD" || got.Name != "Apple Inc." {
		t.Fatalf("unexpected created price: %+v", got)
	}

	if err := runPrice(ctx, c, []string{
		"set", "-ticker", "AAPL", "-date", "2026-01-20", "-price", "105",
	}); err != nil {
		t.Fatalf("price set update: %v", err)
	}
	got, err = c.StockPrices.GetByTickerAndDate(ctx, "AAPL", d)
	if err != nil || got == nil {
		t.Fatalf("get updated price: got=%v err=%v", got, err)
	}
	if got.Price.String() != "105" || got.Currency != "USD" || got.Name != "Apple Inc." {
		t.Fatalf("existing metadata not preserved: %+v", got)
	}
}

func TestPriceSetRejectsUnsupportedCurrencyOnUpdate(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	d, _ := datex.ParseDate("2026-01-20")
	p, _ := numeric.FromString("100")
	if _, err := c.StockPrices.Save(ctx, "AAPL", d, p, "USD", "Apple Inc.", sql.NullString{}); err != nil {
		t.Fatalf("seed price: %v", err)
	}

	if err := runPrice(ctx, c, []string{
		"set", "-ticker", "AAPL", "-date", "2026-01-20", "-price", "105", "-currency", "",
	}); err == nil {
		t.Fatal("expected error when -currency is empty")
	}
	if err := runPrice(ctx, c, []string{
		"set", "-ticker", "AAPL", "-date", "2026-01-20", "-price", "105", "-currency", "EUR",
	}); err == nil {
		t.Fatal("expected error for unsupported currency")
	}

	got, err := c.StockPrices.GetByTickerAndDate(ctx, "AAPL", d)
	if err != nil || got == nil {
		t.Fatalf("get price: got=%v err=%v", got, err)
	}
	if got.Currency != "USD" {
		t.Fatalf("currency was corrupted by rejected update: %+v", got)
	}
}

func TestPriceSetNewRowRequiresMetadata(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	if err := runPrice(ctx, c, []string{
		"set", "-ticker", "AAPL", "-date", "2026-01-20", "-price", "100",
	}); err == nil {
		t.Fatal("expected new cached price to require currency and name")
	}
}

func TestPriceSetUpdatesExistingRowWithEmptyName(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	d, _ := datex.ParseDate("2026-01-20")
	p, _ := numeric.FromString("100")
	if _, err := c.StockPrices.Save(ctx, "AAPL", d, p, "USD", "", sql.NullString{}); err != nil {
		t.Fatalf("seed price: %v", err)
	}
	if err := runPrice(ctx, c, []string{
		"set", "-ticker", "AAPL", "-date", "2026-01-20", "-price", "105",
	}); err != nil {
		t.Fatalf("update existing empty-name row: %v", err)
	}
}

func TestPriceDelete(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	d, _ := datex.ParseDate("2026-01-20")
	p, _ := numeric.FromString("100")
	if _, err := c.StockPrices.Save(ctx, "AAPL", d, p, "USD", "Apple Inc.", sql.NullString{}); err != nil {
		t.Fatalf("seed price: %v", err)
	}
	if err := runPrice(ctx, c, []string{"delete", "-ticker", "AAPL", "-date", "2026-01-20"}); err != nil {
		t.Fatalf("price delete: %v", err)
	}
	got, err := c.StockPrices.GetByTickerAndDate(ctx, "AAPL", d)
	if err != nil {
		t.Fatalf("get deleted price: %v", err)
	}
	if got != nil {
		t.Fatalf("expected deleted price, got %+v", got)
	}
	if err := runPrice(ctx, c, []string{"delete", "-ticker", "AAPL", "-date", "2026-01-20"}); err == nil {
		t.Fatal("expected error for missing cached price")
	}
}

func TestPriceListUnknownVerb(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)

	if err := runPrice(ctx, c, []string{"frobnicate"}); err == nil {
		t.Fatal("expected error for unknown verb")
	}
}

func TestPriceListRejectsFromAfterTo(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)

	err := runPrice(ctx, c, []string{"list", "-ticker", "005930", "-from", "2026-02-01", "-to", "2026-01-01"})
	if err == nil {
		t.Fatal("expected error when -from is after -to")
	}
}
