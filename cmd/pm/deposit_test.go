package main

import (
	"context"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

func newDepositContainer(t *testing.T) *container.Container {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return container.NewWithQueries(sqlDB, q)
}

func firstDepositID(ctx context.Context, t *testing.T, c *container.Container) uuidx.UUID {
	t.Helper()
	deposits, err := c.Deposits.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(deposits) == 0 {
		t.Fatalf("no deposits found")
	}
	return deposits[0].ID
}

func TestDepositList(t *testing.T) {
	c := newDepositContainer(t)
	ctx := context.Background()
	if err := runDeposit(ctx, c, []string{"list"}); err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestDepositGet(t *testing.T) {
	c := newDepositContainer(t)
	ctx := context.Background()
	if err := runDeposit(ctx, c, []string{"add", "-amount", "1000", "-date", "2026-01-15"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	id := firstDepositID(ctx, t, c)
	if err := runDeposit(ctx, c, []string{"get", "-id", id.String()}); err != nil {
		t.Fatalf("deposit get: %v", err)
	}
	if err := runDeposit(ctx, c, []string{"get", "-id", uuidx.New().String()}); err == nil {
		t.Fatal("expected error for unknown id")
	}
}

func TestDepositAddHappyPath(t *testing.T) {
	c := newDepositContainer(t)
	ctx := context.Background()
	if err := runDeposit(ctx, c, []string{"add", "-amount", "1000000", "-date", "2026-01-15", "-note", "salary"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	deposits, err := c.Deposits.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(deposits) != 1 {
		t.Fatalf("want 1 deposit, got %d", len(deposits))
	}
	d := deposits[0]
	if d.Amount.String() != "1000000" {
		t.Errorf("amount = %q, want 1000000", d.Amount.String())
	}
	if d.DepositDate.ISO() != "2026-01-15" {
		t.Errorf("date = %q, want 2026-01-15", d.DepositDate.ISO())
	}
	if !d.Note.Valid || d.Note.String != "salary" {
		t.Errorf("note = %+v, want valid 'salary'", d.Note)
	}
}

func TestDepositAddOnExistingDateUpdates(t *testing.T) {
	c := newDepositContainer(t)
	ctx := context.Background()
	if err := runDeposit(ctx, c, []string{"add", "-amount", "1000000", "-date", "2026-01-15"}); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if err := runDeposit(ctx, c, []string{"add", "-amount", "2500000", "-date", "2026-01-15", "-note", "corrected"}); err != nil {
		t.Fatalf("second add: %v", err)
	}
	deposits, err := c.Deposits.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(deposits) != 1 {
		t.Fatalf("upsert should keep 1 deposit, got %d", len(deposits))
	}
	d := deposits[0]
	if d.Amount.String() != "2500000" {
		t.Errorf("amount = %q, want 2500000", d.Amount.String())
	}
	if !d.Note.Valid || d.Note.String != "corrected" {
		t.Errorf("note = %+v, want valid 'corrected'", d.Note)
	}
}

func TestDepositAddMissingFlags(t *testing.T) {
	c := newDepositContainer(t)
	ctx := context.Background()
	if err := runDeposit(ctx, c, []string{"add", "-date", "2026-01-15"}); err == nil {
		t.Error("add without -amount should error")
	}
	if err := runDeposit(ctx, c, []string{"add", "-amount", "1000000"}); err == nil {
		t.Error("add without -date should error")
	}
}

func TestDepositUpdateAmountOnly(t *testing.T) {
	c := newDepositContainer(t)
	ctx := context.Background()
	if err := runDeposit(ctx, c, []string{"add", "-amount", "1000000", "-date", "2026-01-15", "-note", "keep"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	id := firstDepositID(ctx, t, c)

	if err := runDeposit(ctx, c, []string{"update", "-id", id.String(), "-amount", "3000000"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	d, _ := c.Deposits.GetByID(ctx, id)
	if d.Amount.String() != "3000000" {
		t.Errorf("amount = %q, want 3000000", d.Amount.String())
	}
	if d.DepositDate.ISO() != "2026-01-15" {
		t.Errorf("date changed unexpectedly: %q", d.DepositDate.ISO())
	}
	if !d.Note.Valid || d.Note.String != "keep" {
		t.Errorf("note changed unexpectedly: %+v", d.Note)
	}
}

func TestDepositUpdateDateOnly(t *testing.T) {
	c := newDepositContainer(t)
	ctx := context.Background()
	if err := runDeposit(ctx, c, []string{"add", "-amount", "1000000", "-date", "2026-01-15", "-note", "keep"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	id := firstDepositID(ctx, t, c)

	if err := runDeposit(ctx, c, []string{"update", "-id", id.String(), "-date", "2026-02-20"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	d, _ := c.Deposits.GetByID(ctx, id)
	if d.DepositDate.ISO() != "2026-02-20" {
		t.Errorf("date = %q, want 2026-02-20", d.DepositDate.ISO())
	}
	if d.Amount.String() != "1000000" {
		t.Errorf("amount changed unexpectedly: %q", d.Amount.String())
	}
	if !d.Note.Valid || d.Note.String != "keep" {
		t.Errorf("note changed unexpectedly: %+v", d.Note)
	}
}

func TestDepositUpdateNoteOnly(t *testing.T) {
	c := newDepositContainer(t)
	ctx := context.Background()
	if err := runDeposit(ctx, c, []string{"add", "-amount", "1000000", "-date", "2026-01-15"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	id := firstDepositID(ctx, t, c)

	if err := runDeposit(ctx, c, []string{"update", "-id", id.String(), "-note", "added later"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	d, _ := c.Deposits.GetByID(ctx, id)
	if !d.Note.Valid || d.Note.String != "added later" {
		t.Errorf("note = %+v, want valid 'added later'", d.Note)
	}
	if d.Amount.String() != "1000000" {
		t.Errorf("amount changed unexpectedly: %q", d.Amount.String())
	}
}

func TestDepositUpdateClearingNote(t *testing.T) {
	c := newDepositContainer(t)
	ctx := context.Background()
	if err := runDeposit(ctx, c, []string{"add", "-amount", "1000000", "-date", "2026-01-15", "-note", "remove me"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	id := firstDepositID(ctx, t, c)

	if err := runDeposit(ctx, c, []string{"update", "-id", id.String(), "-note", "/clear"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	d, _ := c.Deposits.GetByID(ctx, id)
	if d.Note.Valid {
		t.Errorf("note should be cleared, got %+v", d.Note)
	}
}

func TestDepositUpdateUnknownID(t *testing.T) {
	c := newDepositContainer(t)
	ctx := context.Background()
	if err := runDeposit(ctx, c, []string{"update", "-id", uuidx.New().String(), "-amount", "5"}); err == nil {
		t.Error("update on unknown id should error")
	}
}

func TestDepositDelete(t *testing.T) {
	c := newDepositContainer(t)
	ctx := context.Background()
	if err := runDeposit(ctx, c, []string{"add", "-amount", "1000000", "-date", "2026-01-15"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	id := firstDepositID(ctx, t, c)

	if err := runDeposit(ctx, c, []string{"delete", "-id", id.String()}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	deposits, _ := c.Deposits.ListAll(ctx)
	if len(deposits) != 0 {
		t.Errorf("want 0 deposits after delete, got %d", len(deposits))
	}
}

func TestDepositDeleteUnknownID(t *testing.T) {
	c := newDepositContainer(t)
	ctx := context.Background()
	if err := runDeposit(ctx, c, []string{"delete", "-id", uuidx.New().String()}); err == nil {
		t.Fatal("expected error for unknown id")
	}
}

func TestDepositUnknownVerb(t *testing.T) {
	c := newDepositContainer(t)
	if err := runDeposit(context.Background(), c, []string{"frobnicate"}); err == nil {
		t.Error("unknown verb should error")
	}
}
