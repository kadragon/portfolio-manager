package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/numeric"
)

func TestRunKisUnknownVerb(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	if err := runKis(ctx, c, []string{"bogus"}); err == nil || !strings.Contains(err.Error(), "unknown kis verb") {
		t.Fatalf("runKis unknown verb error = %v; want unknown kis verb", err)
	}
}

func TestRunKisOrderCashRejectsAccountWithoutKisLink(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	if _, err := c.Accounts.Create(ctx, "manual pension", numeric.Zero); err != nil {
		t.Fatalf("Create: %v", err)
	}

	err := runKis(ctx, c, []string{"order-cash", "-account", "manual pension"})
	if err == nil || !strings.Contains(err.Error(), "has no KIS account number linked") {
		t.Fatalf("runKis order-cash error = %v; want missing KIS link error", err)
	}
}

func TestRunKisOrderCashPriceRequiresTicker(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	// -price without -ticker is a malformed limit query; it must be rejected up
	// front, before any account resolution or KIS call.
	err := runKis(ctx, c, []string{"order-cash", "-account", "isa", "-price", "27470"})
	if err == nil || !strings.Contains(err.Error(), "-price requires -ticker") {
		t.Fatalf("runKis order-cash error = %v; want -price requires -ticker", err)
	}
}

func TestRunKisOrderCashRequiresConfiguredKIS(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	acct, err := c.Accounts.Create(ctx, "isa", numeric.Zero)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Link a KIS account number, but NewWithQueries wires no KIS auth.
	if _, err := c.Accounts.Update(ctx, acct.ID, acct.Name, acct.CashBalance,
		acct.CashBalanceKRW, acct.CashBalanceUSD,
		sql.NullString{String: "12345678-01", Valid: true},
		sql.NullInt64{}, sql.NullString{}, sql.NullInt64{}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	err = runKis(ctx, c, []string{"order-cash", "-account", "isa"})
	if err == nil || !strings.Contains(err.Error(), "KIS is not configured") {
		t.Fatalf("runKis order-cash error = %v; want KIS not configured", err)
	}
}
