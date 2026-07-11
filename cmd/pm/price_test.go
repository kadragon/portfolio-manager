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

func TestPriceListUnknownVerb(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)

	if err := runPrice(ctx, c, []string{"delete"}); err == nil {
		t.Fatal("expected error for unknown verb")
	}
}
