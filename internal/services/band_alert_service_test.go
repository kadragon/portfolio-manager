package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
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

type stubDiagSource struct {
	diags []models.GroupDiagnostic
}

func (s *stubDiagSource) diagnostics(ctx context.Context) ([]models.GroupDiagnostic, error) {
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
