package db

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchema(t *testing.T) {
	s := Schema()
	if s == "" {
		t.Fatal("Schema() returned empty string")
	}
	if !strings.Contains(s, "CREATE TABLE") {
		t.Errorf("Schema() does not contain \"CREATE TABLE\"; got prefix: %.100s", s)
	}
}

func TestDefaultPath_EnvSet(t *testing.T) {
	dir := t.TempDir()
	expected := filepath.Join(dir, "test.db")
	t.Setenv("PORTFOLIO_DB_PATH", expected)

	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath(): %v", err)
	}
	if got != expected {
		t.Errorf("DefaultPath() = %q, want %q", got, expected)
	}
}

func TestDefaultPath_NoEnv(t *testing.T) {
	t.Setenv("PORTFOLIO_DB_PATH", "")

	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath(): %v", err)
	}
	if !strings.Contains(got, ".data/portfolio.db") {
		t.Errorf("DefaultPath() = %q, want it to contain \".data/portfolio.db\"", got)
	}
}

func TestOpenTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	sqlDB, q, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer sqlDB.Close()
	if q == nil {
		t.Fatal("queries is nil")
	}
	// Verify DB is usable
	groups, err := q.ListGroups(context.Background())
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	_ = groups // empty is fine
}

func TestOpenEmptyPath(t *testing.T) {
	dir := t.TempDir()
	tmpDB := filepath.Join(dir, "default.db")
	t.Setenv("PORTFOLIO_DB_PATH", tmpDB)

	sqlDB, q, err := Open("")
	if err != nil {
		t.Fatalf("Open(\"\") with PORTFOLIO_DB_PATH set: %v", err)
	}
	defer sqlDB.Close()
	if q == nil {
		t.Fatal("queries is nil")
	}
}

func TestOpenMemory(t *testing.T) {
	sqlDB, q, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer sqlDB.Close()
	if q == nil {
		t.Fatal("queries is nil")
	}
	groups, err := q.ListGroups(context.Background())
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	_ = groups
}
