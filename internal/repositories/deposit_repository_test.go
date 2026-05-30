package repositories_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
)

func newDepositRepo(t *testing.T) *repositories.DepositRepository {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return repositories.NewDepositRepository(q)
}

func TestDepositCreateAndList(t *testing.T) {
	r := newDepositRepo(t)
	ctx := context.Background()

	amt, _ := numeric.FromString("1000000")
	d1, _ := datex.ParseDate("2026-01-01")
	d2, _ := datex.ParseDate("2026-02-01")
	note := sql.NullString{String: "월급", Valid: true}

	dep1, err := r.Create(ctx, amt, d1, note)
	if err != nil {
		t.Fatalf("create1: %v", err)
	}
	_, err = r.Create(ctx, amt, d2, sql.NullString{})
	if err != nil {
		t.Fatalf("create2: %v", err)
	}

	all, err := r.ListAll(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2, got %d", len(all))
	}
	// ListAll is ordered DESC — d2 (2026-02-01) comes first
	if all[0].DepositDate.ISO() != d2.ISO() {
		t.Errorf("order wrong: [0].date=%s, want %s", all[0].DepositDate.ISO(), d2.ISO())
	}
	if all[1].ID != dep1.ID {
		t.Errorf("id mismatch")
	}
	if !all[1].Note.Valid || all[1].Note.String != "월급" {
		t.Errorf("note mismatch: %+v", all[1].Note)
	}
}

func TestDepositGetByIDAbsentReturnsNil(t *testing.T) {
	r := newDepositRepo(t)
	ctx := context.Background()

	amt, _ := numeric.FromString("500000")
	d, _ := datex.ParseDate("2026-03-01")
	dep, err := r.Create(ctx, amt, d, sql.NullString{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := r.GetByID(ctx, dep.ID)
	if err != nil || got == nil {
		t.Fatalf("get = %v, %v", got, err)
	}

	if err := r.Delete(ctx, dep.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	absent, err := r.GetByID(ctx, dep.ID)
	if err != nil {
		t.Fatalf("get absent err: %v", err)
	}
	if absent != nil {
		t.Fatalf("expected nil after delete")
	}
}

func TestDepositGetByDate(t *testing.T) {
	r := newDepositRepo(t)
	ctx := context.Background()

	amt, _ := numeric.FromString("200000")
	d, _ := datex.ParseDate("2026-04-15")
	created, err := r.Create(ctx, amt, d, sql.NullString{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := r.GetByDate(ctx, d)
	if err != nil || found == nil {
		t.Fatalf("get by date: %v, %v", found, err)
	}
	if found.ID != created.ID {
		t.Fatalf("id mismatch")
	}

	other, _ := datex.ParseDate("2026-04-16")
	none, err := r.GetByDate(ctx, other)
	if err != nil {
		t.Fatalf("get by absent date err: %v", err)
	}
	if none != nil {
		t.Fatalf("expected nil for absent date")
	}
}

func TestDepositUpdateWithNote(t *testing.T) {
	r := newDepositRepo(t)
	ctx := context.Background()

	amt, _ := numeric.FromString("100000")
	d, _ := datex.ParseDate("2026-05-01")
	dep, err := r.Create(ctx, amt, d, sql.NullString{String: "초기 메모", Valid: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newAmt, _ := numeric.FromString("200000")
	newD, _ := datex.ParseDate("2026-05-02")
	newNote := sql.NullString{String: "수정 메모", Valid: true}
	updated, err := r.Update(ctx, dep.ID, newAmt, newD, &newNote)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !updated.Amount.Equal(newAmt.Decimal) {
		t.Fatalf("amount = %v", updated.Amount)
	}
	if updated.DepositDate.ISO() != "2026-05-02" {
		t.Fatalf("date = %s", updated.DepositDate.ISO())
	}
	if !updated.Note.Valid || updated.Note.String != "수정 메모" {
		t.Fatalf("note = %+v", updated.Note)
	}
}

func TestDepositUpdateWithoutNote(t *testing.T) {
	r := newDepositRepo(t)
	ctx := context.Background()

	amt, _ := numeric.FromString("100000")
	d, _ := datex.ParseDate("2026-06-01")
	dep, err := r.Create(ctx, amt, d, sql.NullString{String: "기존 메모", Valid: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newAmt, _ := numeric.FromString("999000")
	// nil note → keep existing
	updated, err := r.Update(ctx, dep.ID, newAmt, d, nil)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !updated.Amount.Equal(newAmt.Decimal) {
		t.Fatalf("amount = %v", updated.Amount)
	}
	if !updated.Note.Valid || updated.Note.String != "기존 메모" {
		t.Fatalf("note changed unexpectedly: %+v", updated.Note)
	}
}

func TestDepositUpdateClearNote(t *testing.T) {
	r := newDepositRepo(t)
	ctx := context.Background()

	amt, _ := numeric.FromString("50000")
	d, _ := datex.ParseDate("2026-07-01")
	dep, err := r.Create(ctx, amt, d, sql.NullString{String: "삭제할 메모", Valid: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	clearNote := sql.NullString{}
	updated, err := r.Update(ctx, dep.ID, amt, d, &clearNote)
	if err != nil {
		t.Fatalf("update clear: %v", err)
	}
	if updated.Note.Valid {
		t.Fatalf("note should be null after clear, got %+v", updated.Note)
	}
}
