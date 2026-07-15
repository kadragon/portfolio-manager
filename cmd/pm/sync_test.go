package main

import (
	"context"
	"strings"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/numeric"
)

func TestRunSyncRejectsAccountWithoutBrokerLink(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	c.KisCano = "12345678"
	c.KisAcntPrdtCd = "01"

	if _, err := c.Accounts.Create(ctx, "manual pension", numeric.Zero); err != nil {
		t.Fatalf("Create: %v", err)
	}

	err := runSync(ctx, c, []string{"-account", "manual pension"})
	if err == nil || !strings.Contains(err.Error(), "has no KIS account number or Toss accountSeq linked") {
		t.Fatalf("runSync error = %v; want missing broker link error", err)
	}
}
