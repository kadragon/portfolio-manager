package main

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

func newStockContainer(t *testing.T) *container.Container {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return container.NewWithQueries(sqlDB, q)
}

func mustGroup(t *testing.T, ctx context.Context, c *container.Container, name string) models.Group {
	t.Helper()
	g, err := c.Groups.Create(ctx, name, 10.0)
	if err != nil {
		t.Fatalf("Groups.Create: %v", err)
	}
	return g
}

func TestStockListAll(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g := mustGroup(t, ctx, c, "Test Group")

	if _, err := c.Stocks.Create(ctx, "AAPL", g.ID); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := c.Stocks.Create(ctx, "MSFT", g.ID); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := runStock(ctx, c, []string{"list"}); err != nil {
		t.Fatalf("stock list: %v", err)
	}
	all, err := c.Stocks.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 stocks, got %d", len(all))
	}
}

func TestStockGet(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g := mustGroup(t, ctx, c, "Test Group")
	s, err := c.Stocks.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := runStock(ctx, c, []string{"get", "-id", s.ID.String()}); err != nil {
		t.Fatalf("stock get: %v", err)
	}
	if err := runStock(ctx, c, []string{"get", "-id", uuidx.New().String()}); err == nil {
		t.Fatal("expected error for unknown id")
	}
}

func TestStockListByGroup(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g1 := mustGroup(t, ctx, c, "Group One")
	g2 := mustGroup(t, ctx, c, "Group Two")

	if _, err := c.Stocks.Create(ctx, "AAPL", g1.ID); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := c.Stocks.Create(ctx, "MSFT", g2.ID); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := runStock(ctx, c, []string{"list", "-group", g1.Name}); err != nil {
		t.Fatalf("stock list -group: %v", err)
	}
	byGroup, err := c.Stocks.ListByGroup(ctx, g1.ID)
	if err != nil {
		t.Fatalf("ListByGroup: %v", err)
	}
	if len(byGroup) != 1 || byGroup[0].Ticker != "AAPL" {
		t.Fatalf("expected only AAPL in g1, got %+v", byGroup)
	}
}

func TestStockAdd(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g := mustGroup(t, ctx, c, "Test Group")

	// lowercase ticker should be uppercased+trimmed
	if err := runStock(ctx, c, []string{"add", "-group", g.ID.String(), "-ticker", "  aapl "}); err != nil {
		t.Fatalf("stock add: %v", err)
	}
	all, err := c.Stocks.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 stock, got %d", len(all))
	}
	if all[0].Ticker != "AAPL" {
		t.Fatalf("expected ticker AAPL, got %q", all[0].Ticker)
	}
	if all[0].GroupID != g.ID {
		t.Fatalf("expected group %s, got %s", g.ID, all[0].GroupID)
	}
}

func TestStockAddMissingGroup(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)

	if err := runStock(ctx, c, []string{"add", "-ticker", "AAPL"}); err == nil {
		t.Fatal("expected error for missing -group")
	}
}

func TestStockUpdateSingleField(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g := mustGroup(t, ctx, c, "Test Group")
	s, err := c.Stocks.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := runStock(ctx, c, []string{"update", "-id", s.ID.String(), "-name", "Apple Inc"}); err != nil {
		t.Fatalf("stock update: %v", err)
	}
	got, err := c.Stocks.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Apple Inc" || got.Ticker != "AAPL" {
		t.Fatalf("after -name update: %+v", got)
	}
}

func TestStockUpdateMultipleFields(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g := mustGroup(t, ctx, c, "Test Group")
	s, err := c.Stocks.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := runStock(ctx, c, []string{
		"update", "-id", s.ID.String(),
		"-ticker", "spy", "-exchange", "NASD", "-asset-class", "etf",
	}); err != nil {
		t.Fatalf("stock update multi: %v", err)
	}
	got, err := c.Stocks.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Ticker != "SPY" {
		t.Fatalf("expected ticker SPY, got %q", got.Ticker)
	}
	if got.Exchange == nil || *got.Exchange != "NASD" {
		t.Fatalf("expected exchange NASD, got %v", got.Exchange)
	}
	if got.AssetClass == nil || *got.AssetClass != "etf" {
		t.Fatalf("expected asset class etf, got %v", got.AssetClass)
	}
}

func TestStockUpdateEmptyTickerRejected(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g := mustGroup(t, ctx, c, "Test Group")
	s, err := c.Stocks.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := runStock(ctx, c, []string{"update", "-id", s.ID.String(), "-ticker", "  "}); err == nil {
		t.Fatal("expected error when -ticker is empty/whitespace")
	}
	got, err := c.Stocks.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Ticker != "AAPL" {
		t.Fatalf("ticker should be unchanged after rejected update, got %q", got.Ticker)
	}
}

func TestStockUpdateInvalidSecurityGroupRejected(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g := mustGroup(t, ctx, c, "Test Group")
	s, err := c.Stocks.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := runStock(ctx, c, []string{"update", "-id", s.ID.String(), "-security-group", "ETF"}); err == nil {
		t.Fatal("expected error for malformed -security-group code")
	}
	got, err := c.Stocks.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.SecurityGroup != nil {
		t.Fatalf("security group should be unchanged after rejected update, got %v", got.SecurityGroup)
	}

	if err := runStock(ctx, c, []string{"update", "-id", s.ID.String(), "-security-group", "EF"}); err != nil {
		t.Fatalf("stock update -security-group: %v", err)
	}
	got, err = c.Stocks.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.SecurityGroup == nil || *got.SecurityGroup != "EF" {
		t.Fatalf("expected security group EF, got %v", got.SecurityGroup)
	}
}

// A code KIS adds after the allowlist was written must still be storable.
func TestStockUpdateUnknownSecurityGroupAcceptedWithWarning(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g := mustGroup(t, ctx, c, "Test Group")
	s, err := c.Stocks.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	originalStderr := os.Stderr
	os.Stderr = writer
	defer func() { os.Stderr = originalStderr }()
	callErr := runStock(ctx, c, []string{"update", "-id", s.ID.String(), "-security-group", "xq"})
	os.Stderr = originalStderr
	if err := writer.Close(); err != nil {
		t.Fatalf("close stderr pipe: %v", err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	if callErr != nil {
		t.Fatalf("stock update -security-group (unknown but well formed): %v", callErr)
	}
	stderr, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if !strings.Contains(string(stderr), "XQ") {
		t.Errorf("expected warning naming the unknown code, got %q", stderr)
	}

	got, err := c.Stocks.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.SecurityGroup == nil || *got.SecurityGroup != "XQ" {
		t.Fatalf("expected security group XQ, got %v", got.SecurityGroup)
	}
}

func TestStockUpdateSecurityGroupNormalizesCase(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g := mustGroup(t, ctx, c, "Test Group")
	s, err := c.Stocks.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := runStock(ctx, c, []string{"update", "-id", s.ID.String(), "-security-group", "  ef  "}); err != nil {
		t.Fatalf("stock update -security-group (lowercase/padded): %v", err)
	}
	got, err := c.Stocks.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.SecurityGroup == nil || *got.SecurityGroup != "EF" {
		t.Fatalf("expected normalized security group EF, got %v", got.SecurityGroup)
	}
}

func TestStockUpdateInvalidSecurityGroupLeavesOtherFieldsUnchanged(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g := mustGroup(t, ctx, c, "Test Group")
	s, err := c.Stocks.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := runStock(ctx, c, []string{
		"update", "-id", s.ID.String(),
		"-ticker", "spy", "-security-group", "ETF",
	}); err == nil {
		t.Fatal("expected error for malformed -security-group code")
	}
	got, err := c.Stocks.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Ticker != "AAPL" {
		t.Fatalf("ticker should be unchanged when -security-group validation fails, got %q", got.Ticker)
	}
	if got.SecurityGroup != nil {
		t.Fatalf("security group should be unchanged, got %v", got.SecurityGroup)
	}
}

func TestStockUpdateNoFields(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g := mustGroup(t, ctx, c, "Test Group")
	s, err := c.Stocks.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := runStock(ctx, c, []string{"update", "-id", s.ID.String()}); err == nil {
		t.Fatal("expected error when no update flags passed")
	}
}

func TestStockMove(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g1 := mustGroup(t, ctx, c, "Group One")
	g2 := mustGroup(t, ctx, c, "Group Two")
	s, err := c.Stocks.Create(ctx, "AAPL", g1.ID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := runStock(ctx, c, []string{"move", "-id", s.ID.String(), "-group", g2.Name}); err != nil {
		t.Fatalf("stock move: %v", err)
	}
	got, err := c.Stocks.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.GroupID != g2.ID {
		t.Fatalf("expected group %s, got %s", g2.ID, got.GroupID)
	}
}

func TestStockDelete(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)
	g := mustGroup(t, ctx, c, "Test Group")
	s, err := c.Stocks.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := runStock(ctx, c, []string{"delete", "-id", s.ID.String()}); err != nil {
		t.Fatalf("stock delete: %v", err)
	}
	got, err := c.Stocks.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected stock deleted, got %+v", got)
	}
}

func TestStockDeleteUnknownID(t *testing.T) {
	ctx := context.Background()
	c := newStockContainer(t)

	if err := runStock(ctx, c, []string{"delete", "-id", uuidx.New().String()}); err == nil {
		t.Fatal("expected error for unknown id")
	}
}
