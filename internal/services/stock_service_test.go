package services_test

import (
	"context"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/services"
)

func newStockServiceContainer(t *testing.T) *container.Container {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return container.NewWithQueries(sqlDB, q)
}

func TestNewStockServiceAndSetPriceService(t *testing.T) {
	c := newStockServiceContainer(t)
	ss := services.NewStockService(c.Stocks, nil)
	if ss == nil {
		t.Fatal("NewStockService returned nil")
	}
	// SetPriceService with nil is a no-op; must not panic
	ss.SetPriceService(nil)
}

func TestPersistNameNoExistingName(t *testing.T) {
	c := newStockServiceContainer(t)
	ctx := context.Background()

	g, err := c.Groups.Create(ctx, "테스트그룹", 100.0)
	if err != nil {
		t.Fatalf("Create group: %v", err)
	}
	stock, err := c.Stocks.Create(ctx, "005930", g.ID)
	if err != nil {
		t.Fatalf("Create stock: %v", err)
	}

	ss := services.NewStockService(c.Stocks, nil)
	result := ss.PersistName(ctx, &stock, "삼성전자")
	if result != "삼성전자" {
		t.Errorf("PersistName = %q, want 삼성전자", result)
	}
	// stock.Name should be updated in place
	if stock.Name != "삼성전자" {
		t.Errorf("stock.Name = %q, want 삼성전자", stock.Name)
	}
}

func TestPersistNameStockAlreadyHasName(t *testing.T) {
	c := newStockServiceContainer(t)
	ctx := context.Background()

	g, err := c.Groups.Create(ctx, "테스트그룹", 100.0)
	if err != nil {
		t.Fatalf("Create group: %v", err)
	}
	stock, err := c.Stocks.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("Create stock: %v", err)
	}
	// Pre-set a name on the stock
	updated, err := c.Stocks.UpdateName(ctx, stock.ID, "Apple Inc.")
	if err != nil {
		t.Fatalf("UpdateName: %v", err)
	}

	ss := services.NewStockService(c.Stocks, nil)
	// rawName takes priority when non-empty
	result := ss.PersistName(ctx, &updated, "Apple Renamed")
	if result != "Apple Renamed" {
		t.Errorf("PersistName = %q, want Apple Renamed", result)
	}
}

func TestResolveAndPersistNameAlreadySet(t *testing.T) {
	c := newStockServiceContainer(t)
	ctx := context.Background()

	g, err := c.Groups.Create(ctx, "테스트그룹", 100.0)
	if err != nil {
		t.Fatalf("Create group: %v", err)
	}
	stock, err := c.Stocks.Create(ctx, "TSLA", g.ID)
	if err != nil {
		t.Fatalf("Create stock: %v", err)
	}
	stock.Name = "Tesla Inc."

	ss := services.NewStockService(c.Stocks, nil)
	result := ss.ResolveAndPersistName(ctx, &stock)
	if result != "Tesla Inc." {
		t.Errorf("ResolveAndPersistName = %q, want Tesla Inc.", result)
	}
}

func TestResolveAndPersistNameNoPriceService(t *testing.T) {
	c := newStockServiceContainer(t)
	ctx := context.Background()

	g, err := c.Groups.Create(ctx, "테스트그룹", 100.0)
	if err != nil {
		t.Fatalf("Create group: %v", err)
	}
	stock, err := c.Stocks.Create(ctx, "VOO", g.ID)
	if err != nil {
		t.Fatalf("Create stock: %v", err)
	}
	// stock.Name is empty; priceService is nil → should return ""

	ss := services.NewStockService(c.Stocks, nil)
	result := ss.ResolveAndPersistName(ctx, &stock)
	if result != "" {
		t.Errorf("ResolveAndPersistName with no priceService = %q, want empty", result)
	}
}
