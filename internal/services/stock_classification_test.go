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
