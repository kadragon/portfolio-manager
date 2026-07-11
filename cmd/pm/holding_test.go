package main

import (
	"context"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

func newHoldingContainer(t *testing.T) *container.Container {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return container.NewWithQueries(sqlDB, q)
}

func holdingFixtures(t *testing.T, ctx context.Context, c *container.Container) (models.Account, models.Stock) {
	t.Helper()
	acc, err := c.Accounts.Create(ctx, "ISA", numeric.Zero)
	if err != nil {
		t.Fatalf("Accounts.Create: %v", err)
	}
	g, err := c.Groups.Create(ctx, "성장주", 100.0)
	if err != nil {
		t.Fatalf("Groups.Create: %v", err)
	}
	stock, err := c.Stocks.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("Stocks.Create: %v", err)
	}
	return acc, stock
}

func mustDecimal(t *testing.T, s string) numeric.Decimal {
	t.Helper()
	d, err := numeric.FromString(s)
	if err != nil {
		t.Fatalf("FromString(%q): %v", s, err)
	}
	return d
}

func TestHoldingListEmptyAndAfterAdd(t *testing.T) {
	ctx := context.Background()
	c := newHoldingContainer(t)
	acc, stock := holdingFixtures(t, ctx, c)

	if err := runHolding(ctx, c, []string{"list", "-account", "ISA"}); err != nil {
		t.Fatalf("holding list (empty): %v", err)
	}
	holdings, err := c.Holdings.ListByAccount(ctx, acc.ID)
	if err != nil {
		t.Fatalf("ListByAccount: %v", err)
	}
	if len(holdings) != 0 {
		t.Fatalf("expected empty, got %d", len(holdings))
	}

	if _, err := c.Holdings.Create(ctx, acc.ID, stock.ID, mustDecimal(t, "10")); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := runHolding(ctx, c, []string{"list", "-account", "ISA"}); err != nil {
		t.Fatalf("holding list: %v", err)
	}
	holdings, err = c.Holdings.ListByAccount(ctx, acc.ID)
	if err != nil {
		t.Fatalf("ListByAccount: %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}
}

func TestHoldingAdd(t *testing.T) {
	ctx := context.Background()
	c := newHoldingContainer(t)
	acc, stock := holdingFixtures(t, ctx, c)

	if err := runHolding(ctx, c, []string{"add", "-account", "ISA", "-stock", stock.ID.String(), "-qty", "12.5"}); err != nil {
		t.Fatalf("holding add: %v", err)
	}
	holdings, err := c.Holdings.ListByAccount(ctx, acc.ID)
	if err != nil {
		t.Fatalf("ListByAccount: %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}
	if holdings[0].StockID != stock.ID || !holdings[0].Quantity.Equal(mustDecimal(t, "12.5").Decimal) {
		t.Fatalf("unexpected holding: %+v", holdings[0])
	}
}

func TestHoldingAddInvalidStock(t *testing.T) {
	ctx := context.Background()
	c := newHoldingContainer(t)
	holdingFixtures(t, ctx, c)

	if err := runHolding(ctx, c, []string{"add", "-account", "ISA", "-stock", "not-a-uuid", "-qty", "1"}); err == nil {
		t.Fatal("expected error for invalid -stock")
	}
}

func TestHoldingAddByTickerExisting(t *testing.T) {
	ctx := context.Background()
	c := newHoldingContainer(t)
	acc, stock := holdingFixtures(t, ctx, c)

	// lowercase ticker exercises normalization to uppercase.
	if err := runHolding(ctx, c, []string{"add-by-ticker", "-account", "ISA", "-ticker", "aapl", "-qty", "3"}); err != nil {
		t.Fatalf("holding add-by-ticker: %v", err)
	}
	holdings, err := c.Holdings.ListByAccount(ctx, acc.ID)
	if err != nil {
		t.Fatalf("ListByAccount: %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}
	if holdings[0].StockID != stock.ID {
		t.Fatalf("expected stock %s, got %s", stock.ID, holdings[0].StockID)
	}
}

func TestHoldingAddByTickerNonexistent(t *testing.T) {
	ctx := context.Background()
	c := newHoldingContainer(t)
	acc, _ := holdingFixtures(t, ctx, c)

	if err := runHolding(ctx, c, []string{"add-by-ticker", "-account", "ISA", "-ticker", "NVDA", "-qty", "1"}); err == nil {
		t.Fatal("expected error for nonexistent ticker")
	}
	holdings, err := c.Holdings.ListByAccount(ctx, acc.ID)
	if err != nil {
		t.Fatalf("ListByAccount: %v", err)
	}
	if len(holdings) != 0 {
		t.Fatalf("expected no holding created, got %d", len(holdings))
	}
}

func TestHoldingBulk(t *testing.T) {
	ctx := context.Background()
	c := newHoldingContainer(t)
	acc, stock := holdingFixtures(t, ctx, c)

	g2, err := c.Groups.Create(ctx, "가치주", 0.0)
	if err != nil {
		t.Fatalf("Groups.Create: %v", err)
	}
	stock2, err := c.Stocks.Create(ctx, "MSFT", g2.ID)
	if err != nil {
		t.Fatalf("Stocks.Create: %v", err)
	}

	h1, err := c.Holdings.Create(ctx, acc.ID, stock.ID, mustDecimal(t, "1"))
	if err != nil {
		t.Fatalf("Create h1: %v", err)
	}
	h2, err := c.Holdings.Create(ctx, acc.ID, stock2.ID, mustDecimal(t, "2"))
	if err != nil {
		t.Fatalf("Create h2: %v", err)
	}

	updates := h1.ID.String() + ":10," + h2.ID.String() + ":20"
	if err := runHolding(ctx, c, []string{"bulk", "-account", "ISA", "-updates", updates}); err != nil {
		t.Fatalf("holding bulk: %v", err)
	}

	got1, err := c.Holdings.GetByID(ctx, h1.ID)
	if err != nil {
		t.Fatalf("GetByID h1: %v", err)
	}
	got2, err := c.Holdings.GetByID(ctx, h2.ID)
	if err != nil {
		t.Fatalf("GetByID h2: %v", err)
	}
	if !got1.Quantity.Equal(mustDecimal(t, "10").Decimal) || !got2.Quantity.Equal(mustDecimal(t, "20").Decimal) {
		t.Fatalf("bulk update not applied: h1=%s h2=%s", got1.Quantity, got2.Quantity)
	}
}

func TestHoldingBulkMalformed(t *testing.T) {
	ctx := context.Background()
	c := newHoldingContainer(t)
	holdingFixtures(t, ctx, c)

	cases := [][]string{
		{"bulk", "-account", "ISA", "-updates", ""},
		{"bulk", "-account", "ISA", "-updates", "no-colon-here"},
		{"bulk", "-account", "ISA", "-updates", "not-a-uuid:10"},
		{"bulk", "-account", "ISA", "-updates", uuidx.New().String() + ":not-a-number"},
	}
	for _, args := range cases {
		if err := runHolding(ctx, c, args); err == nil {
			t.Fatalf("expected error for args %v", args)
		}
	}
}

func TestHoldingUpdate(t *testing.T) {
	ctx := context.Background()
	c := newHoldingContainer(t)
	acc, stock := holdingFixtures(t, ctx, c)

	h, err := c.Holdings.Create(ctx, acc.ID, stock.ID, mustDecimal(t, "1"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := runHolding(ctx, c, []string{"update", "-id", h.ID.String(), "-qty", "99"}); err != nil {
		t.Fatalf("holding update: %v", err)
	}
	got, err := c.Holdings.GetByID(ctx, h.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !got.Quantity.Equal(mustDecimal(t, "99").Decimal) {
		t.Fatalf("expected qty 99, got %s", got.Quantity)
	}
}

func TestHoldingUpdateUnknownID(t *testing.T) {
	ctx := context.Background()
	c := newHoldingContainer(t)
	holdingFixtures(t, ctx, c)

	if err := runHolding(ctx, c, []string{"update", "-id", uuidx.New().String(), "-qty", "1"}); err == nil {
		t.Fatal("expected error for unknown id")
	}
}

func TestHoldingDelete(t *testing.T) {
	ctx := context.Background()
	c := newHoldingContainer(t)
	acc, stock := holdingFixtures(t, ctx, c)

	h, err := c.Holdings.Create(ctx, acc.ID, stock.ID, mustDecimal(t, "1"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := runHolding(ctx, c, []string{"delete", "-id", h.ID.String()}); err != nil {
		t.Fatalf("holding delete: %v", err)
	}
	got, err := c.Holdings.GetByID(ctx, h.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected holding deleted, got %+v", got)
	}
}

func TestHoldingDeleteUnknownID(t *testing.T) {
	ctx := context.Background()
	c := newHoldingContainer(t)
	holdingFixtures(t, ctx, c)

	if err := runHolding(ctx, c, []string{"delete", "-id", uuidx.New().String()}); err != nil {
		t.Fatalf("delete unknown id should be a no-op, got: %v", err)
	}
}

func TestHoldingUnknownVerb(t *testing.T) {
	ctx := context.Background()
	c := newHoldingContainer(t)

	if err := runHolding(ctx, c, []string{"frobnicate"}); err == nil {
		t.Fatal("expected error for unknown verb")
	}
	if err := runHolding(ctx, c, nil); err == nil {
		t.Fatal("expected error for no args")
	}
}
