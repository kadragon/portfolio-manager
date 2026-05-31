package db

import (
	"context"
	"os"
	"testing"
)

// TestProductionDBParity opens a real Peewee-written database and reads rows
// through the sqlc-generated queries and custom SQL types, proving the Go layer
// round-trips data the Python app produced. It is skipped unless
// PORTFOLIO_PARITY_DB points at a copy of such a database (never present in CI).
func TestProductionDBParity(t *testing.T) {
	path := os.Getenv("PORTFOLIO_PARITY_DB")
	if path == "" {
		t.Skip("set PORTFOLIO_PARITY_DB to a Peewee-written SQLite copy to run")
	}
	sqlDB, q, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()

	groups, err := q.ListGroups(context.Background())
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(groups) == 0 {
		t.Fatal("expected groups in production DB, got 0")
	}
	for _, g := range groups {
		if g.ID.UUID.String() == "00000000-0000-0000-0000-000000000000" {
			t.Errorf("group %q has nil UUID", g.Name)
		}
		if g.CreatedAt.Time.IsZero() {
			t.Errorf("group %q has zero created_at", g.Name)
		}
		t.Logf("group: id=%s name=%q target=%.1f created=%s",
			g.ID.Hex(), g.Name, g.TargetPercentage, g.CreatedAt.String())
	}
}
