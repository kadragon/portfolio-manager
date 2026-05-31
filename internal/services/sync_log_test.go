package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

func TestLogEventNoPathIsNoop(t *testing.T) {
	s := &KisAccountSyncService{logPath: ""}
	// Must not panic and must not create any file.
	s.logEvent(map[string]any{"k": "v"}, nil)
}

func TestLogEventWritesLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sync.log")
	s := &KisAccountSyncService{logPath: path}

	s.logEvent(map[string]any{"event": "sync"}, map[string]any{"n": 1})
	s.logEvent(map[string]any{"event": "sync"}, map[string]any{"n": 2})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if lines != 2 {
		t.Errorf("log line count = %d, want 2 (content=%q)", lines, data)
	}
}

func TestRotateIfNeeded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sync.log")

	// Missing file → no-op, no error, no .1 created.
	(&KisAccountSyncService{logPath: path}).rotateIfNeeded()

	// Small file → no rotation.
	if err := os.WriteFile(path, []byte("small"), 0o600); err != nil {
		t.Fatal(err)
	}
	(&KisAccountSyncService{logPath: path}).rotateIfNeeded()
	if _, err := os.Stat(path + ".1"); err == nil {
		t.Errorf("small file should not rotate, but %s.1 exists", path)
	}

	// File at/over the threshold → rotated to .1 (sparse via Truncate).
	if err := os.Truncate(path, _maxSyncLogBytes); err != nil {
		t.Fatal(err)
	}
	(&KisAccountSyncService{logPath: path}).rotateIfNeeded()
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Errorf("oversized file should rotate to %s.1: %v", path, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("after rotation original path should be gone, got err=%v", err)
	}
}

// localGroupRepo is a white-box stub for syncGroupRepo. It lives in
// package services (not services_test), so it cannot reuse the black-box mock.
type localGroupRepo struct {
	existing []models.Group
	created  []models.Group
}

func (r *localGroupRepo) ListAll(_ context.Context) ([]models.Group, error) {
	return r.existing, nil
}

func (r *localGroupRepo) Create(_ context.Context, name string, _ float64) (models.Group, error) {
	g := models.Group{ID: uuidx.New(), Name: name}
	r.created = append(r.created, g)
	return g, nil
}

func TestGetOrCreateSyncGroupID(t *testing.T) {
	ctx := context.Background()

	// Existing default-named group → returned without creating.
	existing := uuidx.New()
	repo := &localGroupRepo{existing: []models.Group{{ID: existing, Name: _defaultSyncGroup}}}
	withGroup := &KisAccountSyncService{defaultGroupName: _defaultSyncGroup, groups: repo}
	id, err := withGroup.getOrCreateSyncGroupID(ctx)
	if err != nil {
		t.Fatalf("existing group: %v", err)
	}
	if id != existing {
		t.Errorf("getOrCreateSyncGroupID = %v, want existing %v", id, existing)
	}
	if len(repo.created) != 0 {
		t.Errorf("should not create when group exists, created=%d", len(repo.created))
	}

	// No matching group → Create path.
	repo2 := &localGroupRepo{existing: []models.Group{{ID: uuidx.New(), Name: "기타"}}}
	noGroup := &KisAccountSyncService{defaultGroupName: _defaultSyncGroup, groups: repo2}
	id2, err := noGroup.getOrCreateSyncGroupID(ctx)
	if err != nil {
		t.Fatalf("create path: %v", err)
	}
	if (id2 == uuidx.UUID{}) {
		t.Errorf("created group ID should be non-zero")
	}
	if len(repo2.created) != 1 || repo2.created[0].Name != _defaultSyncGroup {
		t.Errorf("expected one created group named %q, got %+v", _defaultSyncGroup, repo2.created)
	}
}
