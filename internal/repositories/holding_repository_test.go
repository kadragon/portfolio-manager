package repositories_test

import (
	"context"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

func newHoldingRepo(t *testing.T) (*repositories.AccountRepository, *repositories.StockRepository, *repositories.GroupRepository, *repositories.HoldingRepository) {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return repositories.NewAccountRepository(q),
		repositories.NewStockRepository(q),
		repositories.NewGroupRepository(q),
		repositories.NewHoldingRepository(q)
}

func seedHolding(t *testing.T, accountRepo *repositories.AccountRepository, stockRepo *repositories.StockRepository, groupRepo *repositories.GroupRepository, holdingRepo *repositories.HoldingRepository) (uuidx.UUID, uuidx.UUID, uuidx.UUID) {
	t.Helper()
	ctx := context.Background()
	g, err := groupRepo.Create(ctx, "테스트 그룹", 0)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	s, err := stockRepo.Create(ctx, "005930", g.ID)
	if err != nil {
		t.Fatalf("create stock: %v", err)
	}
	a, err := accountRepo.Create(ctx, "테스트 계좌", numeric.Zero)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	qty, _ := numeric.FromString("10")
	h, err := holdingRepo.Create(ctx, a.ID, s.ID, qty)
	if err != nil {
		t.Fatalf("create holding: %v", err)
	}
	return a.ID, s.ID, h.ID
}

func TestHoldingCreateAndList(t *testing.T) {
	ar, sr, gr, hr := newHoldingRepo(t)
	ctx := context.Background()

	aid, sid, hid := seedHolding(t, ar, sr, gr, hr)

	all, err := hr.ListByAccount(ctx, aid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("want 1 holding, got %d", len(all))
	}
	h := all[0]
	if h.ID != hid {
		t.Fatalf("id mismatch")
	}
	if h.AccountID != aid {
		t.Fatalf("account_id mismatch")
	}
	if h.StockID != sid {
		t.Fatalf("stock_id mismatch")
	}
	qty, _ := numeric.FromString("10")
	if !h.Quantity.Equal(qty.Decimal) {
		t.Fatalf("quantity = %v", h.Quantity)
	}
}

func TestHoldingGetByIDAbsentReturnsNil(t *testing.T) {
	ar, sr, gr, hr := newHoldingRepo(t)
	ctx := context.Background()

	aid, _, hid := seedHolding(t, ar, sr, gr, hr)

	got, err := hr.GetByID(ctx, hid)
	if err != nil || got == nil {
		t.Fatalf("get = %v, %v", got, err)
	}

	if err := hr.Delete(ctx, hid); err != nil {
		t.Fatalf("delete: %v", err)
	}
	absent, err := hr.GetByID(ctx, hid)
	if err != nil {
		t.Fatalf("get absent err: %v", err)
	}
	if absent != nil {
		t.Fatalf("expected nil after delete")
	}
	_ = aid
}

func TestHoldingUpdate(t *testing.T) {
	ar, sr, gr, hr := newHoldingRepo(t)
	ctx := context.Background()

	_, _, hid := seedHolding(t, ar, sr, gr, hr)

	newQty, _ := numeric.FromString("99.5")
	updated, err := hr.Update(ctx, hid, newQty)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !updated.Quantity.Equal(newQty.Decimal) {
		t.Fatalf("quantity = %v, want %v", updated.Quantity, newQty)
	}
}

func TestHoldingBulkUpdateOK(t *testing.T) {
	ar, sr, gr, hr := newHoldingRepo(t)
	ctx := context.Background()

	aid, _, hid := seedHolding(t, ar, sr, gr, hr)

	newQty, _ := numeric.FromString("50")
	updates := []repositories.HoldingUpdate{{ID: hid, Quantity: newQty}}
	if err := hr.BulkUpdateByAccount(ctx, aid, updates); err != nil {
		t.Fatalf("bulk update: %v", err)
	}

	h, err := hr.GetByID(ctx, hid)
	if err != nil || h == nil {
		t.Fatalf("get after bulk: %v, %v", h, err)
	}
	if !h.Quantity.Equal(newQty.Decimal) {
		t.Fatalf("quantity = %v, want %v", h.Quantity, newQty)
	}
}

func TestHoldingBulkUpdateWrongAccount(t *testing.T) {
	ar, sr, gr, hr := newHoldingRepo(t)
	ctx := context.Background()

	_, _, hid := seedHolding(t, ar, sr, gr, hr)

	// Create a second account — try to update holding with wrong account ID
	other, err := ar.Create(ctx, "다른 계좌", numeric.Zero)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	qty, _ := numeric.FromString("1")
	err = hr.BulkUpdateByAccount(ctx, other.ID, []repositories.HoldingUpdate{{ID: hid, Quantity: qty}})
	if err == nil {
		t.Fatal("expected error for wrong account, got nil")
	}
	if err.Error() != "all holdings must belong to account" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHoldingBulkUpdateEmpty(t *testing.T) {
	ar, sr, gr, hr := newHoldingRepo(t)
	ctx := context.Background()

	aid, _, _ := seedHolding(t, ar, sr, gr, hr)

	if err := hr.BulkUpdateByAccount(ctx, aid, nil); err != nil {
		t.Fatalf("empty bulk update: %v", err)
	}
}

func TestHoldingListAll(t *testing.T) {
	ar, sr, gr, hr := newHoldingRepo(t)
	ctx := context.Background()

	aid1, sid1, _ := seedHolding(t, ar, sr, gr, hr)

	g2, _ := gr.Create(ctx, "그룹2", 0)
	s2, _ := sr.Create(ctx, "AAPL", g2.ID)
	acc2, _ := ar.Create(ctx, "계좌2", numeric.Zero)
	_, _ = hr.Create(ctx, acc2.ID, s2.ID, numeric.FromInt(5))
	_ = sid1
	_ = aid1

	all, err := hr.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) < 2 {
		t.Errorf("expected ≥2 holdings, got %d", len(all))
	}
}

func TestHoldingGetAggregatedByStock(t *testing.T) {
	ar, sr, gr, hr := newHoldingRepo(t)
	ctx := context.Background()

	g, _ := gr.Create(ctx, "그룹", 0)
	s, _ := sr.Create(ctx, "005930", g.ID)
	acc1, _ := ar.Create(ctx, "계좌1", numeric.Zero)
	acc2, _ := ar.Create(ctx, "계좌2", numeric.Zero)
	_, _ = hr.Create(ctx, acc1.ID, s.ID, numeric.FromInt(10))
	_, _ = hr.Create(ctx, acc2.ID, s.ID, numeric.FromInt(5))

	agg, err := hr.GetAggregatedByStock(ctx)
	if err != nil {
		t.Fatalf("GetAggregatedByStock: %v", err)
	}
	total, ok := agg[s.ID]
	if !ok {
		t.Fatal("stock not in aggregated map")
	}
	if total.String() != "15" {
		t.Errorf("aggregated qty = %q, want 15", total.String())
	}
}
