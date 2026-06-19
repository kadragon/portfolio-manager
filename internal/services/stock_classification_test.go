package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/services"
)

func TestStockClassificationServiceClassifyAll(t *testing.T) {
	etf := "etf"
	s1 := newTestUUID()
	s2 := newTestUUID()
	s3 := newTestUUID()
	repo := &mockSyncStockRepo{all: []models.Stock{
		{ID: s1, Ticker: "0052D0"},                   // domestic, nil → classify
		{ID: s2, Ticker: "SCHD"},                     // overseas, nil → classify
		{ID: s3, Ticker: "005930", AssetClass: &etf}, // already classified → skip
	}}
	classifier := &fakeAssetClassifier{byTicker: map[string]string{"0052D0": "etf", "SCHD": "etf"}}
	svc := services.NewStockClassificationService(repo, classifier)

	if !svc.Enabled() {
		t.Fatal("want Enabled() true with classifier set")
	}
	res, err := svc.ClassifyAll(context.Background())
	if err != nil {
		t.Fatalf("ClassifyAll: %v", err)
	}
	if res.Total != 3 || res.Classified != 2 || res.Skipped != 1 || res.Failed != 0 {
		t.Errorf("result = %+v, want {Total:3 Classified:2 Skipped:1 Failed:0}", res)
	}
	if len(repo.classified) != 2 {
		t.Errorf("persisted classifications = %d, want 2", len(repo.classified))
	}
}

func TestStockClassificationServiceBackfillsSecurityGroup(t *testing.T) {
	etf := "etf"
	s1 := newTestUUID() // unclassified: gets both asset_class + security_group
	s2 := newTestUUID() // asset_class set, security_group nil: gets only security_group
	repo := &mockSyncStockRepo{all: []models.Stock{
		{ID: s1, Ticker: "0052D0"},
		{ID: s2, Ticker: "069500", AssetClass: &etf},
	}}
	classifier := &fakeAssetClassifier{
		byTicker:   map[string]string{"0052D0": "etf", "069500": "etf"},
		bySecGroup: map[string]string{"0052D0": "EF", "069500": "EF"},
	}
	svc := services.NewStockClassificationService(repo, classifier)

	res, err := svc.ClassifyAll(context.Background())
	if err != nil {
		t.Fatalf("ClassifyAll: %v", err)
	}
	if res.Total != 2 || res.Classified != 2 || res.Failed != 0 {
		t.Errorf("result = %+v, want {Total:2 Classified:2 Failed:0}", res)
	}
	if len(repo.secGrouped) != 2 {
		t.Errorf("security_group updates = %d, want 2", len(repo.secGrouped))
	}
	// s2 already had asset_class; only security_group should be written for it.
	if len(repo.classified) != 1 {
		t.Errorf("asset_class updates = %d, want 1 (s2 was already classified)", len(repo.classified))
	}
}

func TestStockClassificationServiceDisabled(t *testing.T) {
	svc := services.NewStockClassificationService(&mockSyncStockRepo{}, nil)
	if svc.Enabled() {
		t.Error("want Enabled() false for nil classifier")
	}
}

func TestStockClassificationServiceCountsFailure(t *testing.T) {
	repo := &mockSyncStockRepo{all: []models.Stock{{ID: newTestUUID(), Ticker: "0052D0"}}}
	classifier := &fakeAssetClassifier{err: errors.New("kis down")}
	svc := services.NewStockClassificationService(repo, classifier)

	res, err := svc.ClassifyAll(context.Background())
	if err != nil {
		t.Fatalf("ClassifyAll returned hard error: %v", err)
	}
	if res.Failed != 1 || res.Classified != 0 {
		t.Errorf("result = %+v, want Failed:1 Classified:0", res)
	}
}

// On classifier failure the stock is sentinel-tagged ("unknown") on BOTH
// asset_class and security_group so the skip-gates stop re-querying it.
func TestClassifyAllPersistsSentinelOnFailure(t *testing.T) {
	id := newTestUUID()
	repo := &mockSyncStockRepo{all: []models.Stock{{ID: id, Ticker: "BADTICK"}}}
	classifier := &fakeAssetClassifier{err: errors.New("kis down")}
	svc := services.NewStockClassificationService(repo, classifier)

	if _, err := svc.ClassifyAll(context.Background()); err != nil {
		t.Fatalf("ClassifyAll: %v", err)
	}
	if len(repo.classified) != 1 || repo.classified[0].assetClass != services.AssetClassUnknown {
		t.Errorf("asset_class updates = %+v, want one %q", repo.classified, services.AssetClassUnknown)
	}
	if len(repo.secGrouped) != 1 || repo.secGrouped[0].securityGroup != services.AssetClassUnknown {
		t.Errorf("security_group updates = %+v, want one %q", repo.secGrouped, services.AssetClassUnknown)
	}
}

// A classifier that returns no signal (empty strings, no error) also yields the
// sentinel — there is nothing to retry on.
func TestClassifyAllPersistsSentinelOnEmpty(t *testing.T) {
	id := newTestUUID()
	repo := &mockSyncStockRepo{all: []models.Stock{{ID: id, Ticker: "UNKWN"}}}
	classifier := &fakeAssetClassifier{byTicker: map[string]string{}} // returns "", "", nil
	svc := services.NewStockClassificationService(repo, classifier)

	if _, err := svc.ClassifyAll(context.Background()); err != nil {
		t.Fatalf("ClassifyAll: %v", err)
	}
	if len(repo.classified) != 1 || repo.classified[0].assetClass != services.AssetClassUnknown {
		t.Errorf("asset_class updates = %+v, want one %q", repo.classified, services.AssetClassUnknown)
	}
}

// A stock already carrying the sentinel on both columns is skipped — the
// classifier is never called again.
func TestClassifyAllSkipsSentinelStock(t *testing.T) {
	unknown := services.AssetClassUnknown
	repo := &mockSyncStockRepo{all: []models.Stock{
		{ID: newTestUUID(), Ticker: "BADTICK", AssetClass: &unknown, SecurityGroup: &unknown},
	}}
	classifier := &fakeAssetClassifier{}
	svc := services.NewStockClassificationService(repo, classifier)

	res, err := svc.ClassifyAll(context.Background())
	if err != nil {
		t.Fatalf("ClassifyAll: %v", err)
	}
	if classifier.calls != 0 {
		t.Errorf("classifier called %d times, want 0 (sentinel skipped)", classifier.calls)
	}
	if res.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", res.Skipped)
	}
}

// ClassifyAll honors context cancellation — a cancelled context stops the loop
// before issuing KIS calls.
func TestClassifyAllHonorsContextCancellation(t *testing.T) {
	repo := &mockSyncStockRepo{all: []models.Stock{{ID: newTestUUID(), Ticker: "0052D0"}}}
	classifier := &fakeAssetClassifier{byTicker: map[string]string{"0052D0": "etf"}}
	svc := services.NewStockClassificationService(repo, classifier)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := svc.ClassifyAll(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if classifier.calls != 0 {
		t.Errorf("classifier called %d times after cancel, want 0", classifier.calls)
	}
}
