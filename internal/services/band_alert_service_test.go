package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
	"github.com/shopspring/decimal"
)

func diag(name string, upper, lower bool) models.GroupDiagnostic {
	return models.GroupDiagnostic{
		RebalanceGroupName: name,
		TargetPct:          numeric.Wrap(decimal.RequireFromString("35")),
		BandPct:            numeric.Wrap(decimal.RequireFromString("5")),
		CurrentPct:         numeric.Wrap(decimal.RequireFromString("45")),
		IsUpperBreached:    upper,
		IsLowerBreached:    lower,
	}
}

// TestBreachSignature: signature includes only breached groups with direction,
// in _groupOrder order, so it changes exactly when the breach set changes.
func TestBreachSignature(t *testing.T) {
	diags := []models.GroupDiagnostic{
		diag("국내성장", true, false),
		diag("국내배당", false, false),
		diag("해외배당", false, true),
	}
	sig := breachSignature(diags)
	if sig != "국내성장:상단,해외배당:하단" {
		t.Errorf("signature = %q, want 국내성장:상단,해외배당:하단", sig)
	}
	if got := breachSignature([]models.GroupDiagnostic{diag("국내성장", false, false)}); got != "" {
		t.Errorf("no breach must yield empty signature, got %q", got)
	}
}

// TestBandAlertErrorOmitsWebhookURL: webhook URLs carry secret tokens
// (Slack/Discord); a failed request's error — which gets logged — must not
// echo the URL back.
func TestBandAlertErrorOmitsWebhookURL(t *testing.T) {
	secretURL := "http://127.0.0.1:1/services/SECRET-TOKEN"
	svc := &BandAlertService{
		source:     &stubDiagSource{diags: []models.GroupDiagnostic{diag("국내성장", true, false)}},
		webhookURL: secretURL,
		client:     &http.Client{},
	}
	err := svc.CheckOnce(context.Background())
	if err == nil {
		t.Fatal("unreachable webhook must error")
	}
	if strings.Contains(err.Error(), "SECRET-TOKEN") {
		t.Errorf("error leaks webhook URL: %v", err)
	}
}

type stubDiagSource struct {
	diags []models.GroupDiagnostic
	calls int
}

func (s *stubDiagSource) diagnostics(ctx context.Context) ([]models.GroupDiagnostic, error) {
	s.calls++
	return s.diags, nil
}

// TestBandAlertNotifiesOnceUntilBreachSetChanges: a persisting breach alerts on
// first sight only; clearing resets; a new breach alerts again.
func TestBandAlertNotifiesOnceUntilBreachSetChanges(t *testing.T) {
	var calls int
	var lastBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, _ := io.ReadAll(r.Body)
		lastBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	source := &stubDiagSource{diags: []models.GroupDiagnostic{diag("국내성장", true, false)}}
	svc := &BandAlertService{source: source, webhookURL: srv.URL, client: srv.Client()}
	ctx := context.Background()

	if err := svc.CheckOnce(ctx); err != nil {
		t.Fatalf("CheckOnce: %v", err)
	}
	if calls != 1 {
		t.Fatalf("first breach must notify, calls = %d", calls)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(lastBody), &payload); err != nil {
		t.Fatalf("payload not JSON: %v (%s)", err, lastBody)
	}
	if !strings.Contains(payload["text"], "국내성장") || payload["content"] != payload["text"] {
		t.Errorf("payload must carry breach text in both text/content keys, got %s", lastBody)
	}

	// Same breach persists → no re-notify.
	if err := svc.CheckOnce(ctx); err != nil {
		t.Fatalf("CheckOnce: %v", err)
	}
	if calls != 1 {
		t.Errorf("persisting breach must not re-notify, calls = %d", calls)
	}

	// Breach clears → no notify, state resets.
	source.diags = []models.GroupDiagnostic{diag("국내성장", false, false)}
	if err := svc.CheckOnce(ctx); err != nil {
		t.Fatalf("CheckOnce: %v", err)
	}
	if calls != 1 {
		t.Errorf("cleared breach must not notify, calls = %d", calls)
	}

	// Re-breach → notify again.
	source.diags = []models.GroupDiagnostic{diag("국내성장", true, false)}
	if err := svc.CheckOnce(ctx); err != nil {
		t.Fatalf("CheckOnce: %v", err)
	}
	if calls != 2 {
		t.Errorf("new breach after clear must notify, calls = %d", calls)
	}
}

// TestNewBandAlertServiceWiresProductionSource: constructor must produce a
// ready-to-run service (source + HTTP client set) from repositories alone.
func TestNewBandAlertServiceWiresProductionSource(t *testing.T) {
	svc := NewBandAlertService(nil, nil, "http://example.invalid/hook")
	if svc.source == nil {
		t.Fatal("source not wired")
	}
	if svc.client == nil || svc.webhookURL == "" {
		t.Fatal("client/webhookURL not wired")
	}
}

// TestStartChecksOnceAndStopsOnCancel: Start performs an immediate check and
// exits promptly when the context is cancelled — the daily ticker must not
// keep the goroutine alive past shutdown.
func TestStartChecksOnceAndStopsOnCancel(t *testing.T) {
	source := &stubDiagSource{diags: nil}
	svc := &BandAlertService{source: source, client: http.DefaultClient}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		svc.Start(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancel")
	}
	if source.calls == 0 {
		t.Fatal("Start must run an immediate check before waiting on the ticker")
	}
}

// TestPortfolioBandSourceDiagnostics: the production source composes
// PortfolioService + GroupRepository; on an empty portfolio it must yield no
// diagnostics and no error.
func TestPortfolioBandSourceDiagnostics(t *testing.T) {
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	groups := repositories.NewGroupRepository(q)
	stocks := repositories.NewStockRepository(q)
	holdings := repositories.NewHoldingRepository(q, sqlDB)
	accounts := repositories.NewAccountRepository(q)
	deposits := repositories.NewDepositRepository(q)
	prices := NewPriceService(repositories.NewStockPriceRepository(q))
	portfolio := NewPortfolioService(groups, stocks, holdings, accounts, deposits, prices, nil)
	if _, err := groups.Create(context.Background(), "금", 10); err != nil {
		t.Fatalf("create group: %v", err)
	}
	src := &portfolioBandSource{portfolio: portfolio, groups: groups}
	diags, err := src.diagnostics(context.Background())
	if err != nil {
		t.Fatalf("diagnostics: %v", err)
	}
	if len(diags) != 0 {
		t.Fatalf("empty portfolio (total 0) must yield no diagnostics, got %d", len(diags))
	}
}
