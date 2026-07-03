package services_test

import (
	"testing"

	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/services"
)

// TestCheckBandsReturnsDiagnosticsWithoutPlan: CheckBands yields the same
// band-state diagnostics BuildPlan would, from summary + groups alone — no
// accounts or per-account holdings needed (used by the band-alert scheduler).
func TestCheckBandsReturnsDiagnosticsWithoutPlan(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	// total 990; 국내성장 500 = 50.5% > upper 40 → breached; others in band.
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("500"),
		"국내배당": mustN("100"),
		"해외성장": mustN("200"),
		"해외안정": mustN("90"),
		"해외배당": mustN("100"),
	})

	svc := services.NewRebalanceService()
	diags, err := svc.CheckBands(summary, groups)
	if err != nil {
		t.Fatalf("CheckBands error: %v", err)
	}
	if len(diags) != 5 {
		t.Fatalf("want 5 diagnostics, got %d", len(diags))
	}
	byName := map[string]bool{}
	for _, d := range diags {
		byName[d.RebalanceGroupName] = d.IsUpperBreached
	}
	if !byName["국내성장"] {
		t.Errorf("국내성장 at 50.5%% must be upper-breached")
	}
	for _, g := range []string{"국내배당", "해외성장", "해외안정", "해외배당"} {
		if byName[g] {
			t.Errorf("%s must not be breached", g)
		}
	}
}
