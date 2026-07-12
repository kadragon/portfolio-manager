package main

import (
	"context"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

func newGroupContainer(t *testing.T) *container.Container {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return container.NewWithQueries(sqlDB, q)
}

func TestGroupList(t *testing.T) {
	ctx := context.Background()
	c := newGroupContainer(t)

	groups, err := c.Groups.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected empty, got %d", len(groups))
	}
	if err := runGroup(ctx, c, []string{"list"}); err != nil {
		t.Fatalf("group list (empty): %v", err)
	}

	if _, err := c.Groups.Create(ctx, "성장주", 60.0); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := runGroup(ctx, c, []string{"list"}); err != nil {
		t.Fatalf("group list: %v", err)
	}
	groups, err = c.Groups.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
}

func TestGroupGet(t *testing.T) {
	ctx := context.Background()
	c := newGroupContainer(t)
	g, err := c.Groups.Create(ctx, "성장주", 60)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := runGroup(ctx, c, []string{"get", "-id", g.ID.String()}); err != nil {
		t.Fatalf("group get: %v", err)
	}
	if err := runGroup(ctx, c, []string{"get", "-id", uuidx.New().String()}); err == nil {
		t.Fatal("expected error for unknown id")
	}
}

func TestGroupAdd(t *testing.T) {
	ctx := context.Background()
	c := newGroupContainer(t)

	if err := runGroup(ctx, c, []string{"add", "-name", "가치주", "-target", "25.0"}); err != nil {
		t.Fatalf("group add: %v", err)
	}
	groups, err := c.Groups.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Name != "가치주" || groups[0].TargetPercentage != 25.0 {
		t.Fatalf("unexpected group: %+v", groups[0])
	}
}

func TestGroupAddMissingName(t *testing.T) {
	ctx := context.Background()
	c := newGroupContainer(t)

	if err := runGroup(ctx, c, []string{"add", "-target", "25.0"}); err == nil {
		t.Fatal("expected error for missing -name")
	}
}

func TestGroupUpdatePartial(t *testing.T) {
	ctx := context.Background()
	c := newGroupContainer(t)

	g, err := c.Groups.Create(ctx, "old", 10.0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id := g.ID.String()

	// only -name
	if err := runGroup(ctx, c, []string{"update", "-id", id, "-name", "new"}); err != nil {
		t.Fatalf("update name: %v", err)
	}
	got, err := c.Groups.GetByID(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "new" || got.TargetPercentage != 10.0 {
		t.Fatalf("after -name update: %+v", got)
	}

	// only -target
	if err := runGroup(ctx, c, []string{"update", "-id", id, "-target", "42.5"}); err != nil {
		t.Fatalf("update target: %v", err)
	}
	got, err = c.Groups.GetByID(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "new" || got.TargetPercentage != 42.5 {
		t.Fatalf("after -target update: %+v", got)
	}

	// both
	if err := runGroup(ctx, c, []string{"update", "-id", id, "-name", "both", "-target", "99.0"}); err != nil {
		t.Fatalf("update both: %v", err)
	}
	got, err = c.Groups.GetByID(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "both" || got.TargetPercentage != 99.0 {
		t.Fatalf("after both update: %+v", got)
	}
}

func TestGroupUpdateUnknownID(t *testing.T) {
	ctx := context.Background()
	c := newGroupContainer(t)

	if err := runGroup(ctx, c, []string{"update", "-id", uuidx.New().String(), "-name", "x"}); err == nil {
		t.Fatal("expected error for unknown id")
	}
}

func TestGroupDelete(t *testing.T) {
	ctx := context.Background()
	c := newGroupContainer(t)

	g, err := c.Groups.Create(ctx, "doomed", 5.0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := runGroup(ctx, c, []string{"delete", "-id", g.ID.String()}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err := c.Groups.GetByID(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected group deleted, got %+v", got)
	}
}

func TestGroupDeleteUnknownID(t *testing.T) {
	ctx := context.Background()
	c := newGroupContainer(t)

	if err := runGroup(ctx, c, []string{"delete", "-id", uuidx.New().String()}); err == nil {
		t.Fatal("expected error for unknown id")
	}
}
