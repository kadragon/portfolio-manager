package repositories_test

import (
	"context"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/repositories"
)

func newGroupRepo(t *testing.T) *repositories.GroupRepository {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return repositories.NewGroupRepository(q)
}

func TestGroupCreateAndList(t *testing.T) {
	repo := newGroupRepo(t)
	ctx := context.Background()

	g, err := repo.Create(ctx, "성장주", 30.0)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if g.Name != "성장주" || g.TargetPercentage != 30.0 {
		t.Fatalf("created = %+v", g)
	}
	if g.ID.UUID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Fatal("created group has nil id")
	}
	if g.CreatedAt.IsZero() || g.UpdatedAt.IsZero() {
		t.Fatal("timestamps not set")
	}

	all, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 || all[0].ID != g.ID {
		t.Fatalf("list = %+v", all)
	}
}

func TestGroupListInsertionOrder(t *testing.T) {
	repo := newGroupRepo(t)
	ctx := context.Background()
	names := []string{"가", "나", "다"}
	for _, n := range names {
		if _, err := repo.Create(ctx, n, 0); err != nil {
			t.Fatalf("create %s: %v", n, err)
		}
	}
	all, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("len = %d", len(all))
	}
	for i, n := range names {
		if all[i].Name != n {
			t.Errorf("position %d = %q, want %q (insertion order)", i, all[i].Name, n)
		}
	}
}

func TestGroupListEmpty(t *testing.T) {
	repo := newGroupRepo(t)
	all, err := repo.ListAll(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected empty, got %d", len(all))
	}
}

func TestGroupGetByIDAbsentReturnsNil(t *testing.T) {
	repo := newGroupRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "배당주", 20.0)
	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil || got.Name != "배당주" {
		t.Fatalf("get = %+v", got)
	}

	_ = repo.Delete(ctx, created.ID)
	absent, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get absent: %v", err)
	}
	if absent != nil {
		t.Fatalf("expected nil for deleted group, got %+v", absent)
	}
}

func TestGroupUpdatePartial(t *testing.T) {
	repo := newGroupRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "old", 10.0)

	// Update only the name; target stays.
	newName := "new"
	updated, err := repo.Update(ctx, created.ID, &newName, nil)
	if err != nil {
		t.Fatalf("update name: %v", err)
	}
	if updated.Name != "new" || updated.TargetPercentage != 10.0 {
		t.Fatalf("partial name update = %+v", updated)
	}

	// Update only the target; name stays.
	newTarget := 55.5
	updated, err = repo.Update(ctx, created.ID, nil, &newTarget)
	if err != nil {
		t.Fatalf("update target: %v", err)
	}
	if updated.Name != "new" || updated.TargetPercentage != 55.5 {
		t.Fatalf("partial target update = %+v", updated)
	}
}

func TestGroupDelete(t *testing.T) {
	repo := newGroupRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "tmp", 0.0)
	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	all, _ := repo.ListAll(ctx)
	if len(all) != 0 {
		t.Fatalf("expected empty after delete, got %d", len(all))
	}
}
