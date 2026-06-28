package repositories_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
)

func newAccountRepo(t *testing.T) *repositories.AccountRepository {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return repositories.NewAccountRepository(q)
}

func TestAccountTypeRoundTrip(t *testing.T) {
	repo := newAccountRepo(t)
	ctx := context.Background()

	a, err := repo.Create(ctx, "IRP", numeric.Zero)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if a.AccountType != nil {
		t.Fatalf("fresh account_type should be nil, got %v", *a.AccountType)
	}

	updated, err := repo.Update(ctx, a.ID, a.Name, a.CashBalance,
		sql.NullString{}, sql.NullInt64{},
		sql.NullString{String: "irp", Valid: true}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.AccountType == nil || *updated.AccountType != "irp" {
		t.Fatalf("account_type = %v, want irp", updated.AccountType)
	}

	got, err := repo.GetByID(ctx, a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AccountType == nil || *got.AccountType != "irp" {
		t.Fatalf("reloaded account_type = %v, want irp", got.AccountType)
	}
}

func TestAccountCreateAndList(t *testing.T) {
	repo := newAccountRepo(t)
	ctx := context.Background()

	a, err := repo.Create(ctx, "테스트 계좌", numeric.Zero)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if a.Name != "테스트 계좌" {
		t.Fatalf("name = %q", a.Name)
	}
	if !a.CashBalance.IsZero() {
		t.Fatalf("cash = %v", a.CashBalance)
	}
	if a.KisAccountNo != nil || a.KisAPIKeyID != nil {
		t.Fatalf("KIS fields should be nil: %+v", a)
	}
	if a.TossAccountSeq != nil {
		t.Fatalf("Toss account seq should be nil: %+v", a)
	}

	all, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 || all[0].ID != a.ID {
		t.Fatalf("list = %+v", all)
	}
}

func TestAccountGetByIDAbsentReturnsNil(t *testing.T) {
	repo := newAccountRepo(t)
	ctx := context.Background()

	a, err := repo.Create(ctx, "X", numeric.Zero)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, a.ID)
	if err != nil || got == nil || got.Name != "X" {
		t.Fatalf("get = %v, %v", got, err)
	}

	if err := repo.DeleteWithHoldings(ctx, a.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	absent, err := repo.GetByID(ctx, a.ID)
	if err != nil {
		t.Fatalf("get absent: %v", err)
	}
	if absent != nil {
		t.Fatalf("expected nil, got %+v", absent)
	}
}

func TestAccountUpdateNameCash(t *testing.T) {
	repo := newAccountRepo(t)
	ctx := context.Background()

	a, err := repo.Create(ctx, "초기", numeric.Zero)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	cash, _ := numeric.FromString("1000000")
	updated, err := repo.UpdateNameCash(ctx, a.ID, "수정됨", cash)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "수정됨" {
		t.Fatalf("name = %q", updated.Name)
	}
	if !updated.CashBalance.Equal(cash.Decimal) {
		t.Fatalf("cash = %v", updated.CashBalance)
	}
	// KIS fields should remain nil
	if updated.KisAccountNo != nil || updated.KisAPIKeyID != nil {
		t.Fatalf("KIS fields changed: %+v", updated)
	}
}

func TestAccountUpdateFull(t *testing.T) {
	repo := newAccountRepo(t)
	ctx := context.Background()

	a, err := repo.Create(ctx, "A", numeric.Zero)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	cash, _ := numeric.FromString("500000")
	updated, err := repo.Update(ctx, a.ID, "B", cash,
		sql.NullString{String: "12345678-01", Valid: true},
		sql.NullInt64{Int64: 1, Valid: true}, sql.NullString{},
		sql.NullInt64{Int64: 7, Valid: true},
	)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "B" {
		t.Fatalf("name = %q", updated.Name)
	}
	if updated.KisAccountNo == nil || *updated.KisAccountNo != "12345678-01" {
		t.Fatalf("kis_account_no = %v", updated.KisAccountNo)
	}
	if updated.KisAPIKeyID == nil || *updated.KisAPIKeyID != 1 {
		t.Fatalf("kis_api_key_id = %v", updated.KisAPIKeyID)
	}
	if updated.TossAccountSeq == nil || *updated.TossAccountSeq != 7 {
		t.Fatalf("toss_account_seq = %v", updated.TossAccountSeq)
	}

	// Clear KIS by setting NULL
	cleared, err := repo.Update(ctx, a.ID, "B", cash,
		sql.NullString{Valid: false},
		sql.NullInt64{Valid: false}, sql.NullString{}, sql.NullInt64{},
	)
	if err != nil {
		t.Fatalf("clear KIS: %v", err)
	}
	if cleared.KisAccountNo != nil || cleared.KisAPIKeyID != nil || cleared.TossAccountSeq != nil {
		t.Fatalf("KIS fields not cleared: %+v", cleared)
	}
}

func TestAccountUpdateNameCashPreservesKIS(t *testing.T) {
	repo := newAccountRepo(t)
	ctx := context.Background()

	a, _ := repo.Create(ctx, "A", numeric.Zero)
	cash, _ := numeric.FromString("1000")
	_, err := repo.Update(ctx, a.ID, "A", cash,
		sql.NullString{String: "12345678-01", Valid: true},
		sql.NullInt64{Int64: 1, Valid: true}, sql.NullString{},
		sql.NullInt64{Int64: 3, Valid: true},
	)
	if err != nil {
		t.Fatalf("set KIS: %v", err)
	}

	newCash, _ := numeric.FromString("9999")
	updated, err := repo.UpdateNameCash(ctx, a.ID, "A", newCash)
	if err != nil {
		t.Fatalf("update name/cash: %v", err)
	}
	if updated.KisAccountNo == nil || *updated.KisAccountNo != "12345678-01" {
		t.Fatalf("KIS account no changed: %v", updated.KisAccountNo)
	}
	if updated.KisAPIKeyID == nil || *updated.KisAPIKeyID != 1 {
		t.Fatalf("KIS key id changed: %v", updated.KisAPIKeyID)
	}
	if updated.TossAccountSeq == nil || *updated.TossAccountSeq != 3 {
		t.Fatalf("Toss account seq changed: %v", updated.TossAccountSeq)
	}
}
