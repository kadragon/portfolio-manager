package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestOrderExecutionList(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	if _, err := c.OrderExecutions.Create(ctx, "AAPL", "buy", 1, "USD", "filled", "ok", "NASD", nil); err != nil {
		t.Fatalf("Create filled: %v", err)
	}
	if _, err := c.OrderExecutions.Create(ctx, "005930", "sell", 2, "KRW", "failed", "rejected", "KRX", map[string]any{"account": "sensitive"}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := runOrderExecution(ctx, c, []string{"list", "-limit", "10"}); err != nil {
		t.Fatalf("order-execution list: %v", err)
	}
	if err := runOrderExecution(ctx, c, []string{"list", "-status", "failed", "-ticker", "005930"}); err != nil {
		t.Fatalf("order-execution filtered list: %v", err)
	}
	records, err := c.OrderExecutions.List(ctx, "005930", "failed", 20)
	if err != nil {
		t.Fatalf("List filtered records: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("filtered record count = %d, want 1", len(records))
	}
	encoded, err := json.Marshal(toOrderExecutionOutputs(records))
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}
	if strings.Contains(string(encoded), "RawResponse") || strings.Contains(string(encoded), "sensitive") {
		t.Fatalf("order execution output exposed raw response: %s", encoded)
	}
}

func TestOrderExecutionListRejectsInvalidLimit(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	if err := runOrderExecution(ctx, c, []string{"list", "-limit", "0"}); err == nil {
		t.Fatal("expected error for non-positive limit")
	}
}
