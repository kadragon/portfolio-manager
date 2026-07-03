package services

import (
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
	"github.com/shopspring/decimal"
)

// bandFixture builds a summary + groups where each group holds one position
// worth valueKRW. Groups come from the DB in production; nothing is hardcoded
// in checkBands, so the fixture defines its own group set.
func bandFixture(t *testing.T, targets map[string]float64, values map[string]string) (models.PortfolioSummary, []models.Group) {
	t.Helper()
	groups := make([]models.Group, 0, len(targets))
	pairs := []models.GroupHoldingPair{}
	total := decimal.Zero
	for name, target := range targets {
		g := models.Group{ID: uuidx.New(), Name: name, TargetPercentage: target}
		groups = append(groups, g)
		v, err := decimal.NewFromString(values[name])
		if err != nil {
			t.Fatalf("bad value %q: %v", values[name], err)
		}
		vKRW := numeric.Wrap(v)
		pairs = append(pairs, models.GroupHoldingPair{
			Group: g,
			Holding: models.StockHoldingWithPrice{
				Stock:    models.Stock{Ticker: name, GroupID: g.ID},
				Quantity: vKRW,
				Price:    numeric.Wrap(decimal.NewFromInt(1)),
				Currency: "KRW",
				Name:     name,
				ValueKRW: &vKRW,
			},
		})
		total = total.Add(v)
	}
	summary := models.PortfolioSummary{
		Holdings:    pairs,
		TotalValue:  numeric.Wrap(total),
		TotalAssets: numeric.Wrap(total),
	}
	return summary, groups
}

// TestCheckBandsAppliesFiveTwentyFiveRule: band = min(5%p, 25% of target).
// 국내성장 target 35 → band 5 (absolute cap); at 50.5% it is upper-breached.
// 채권 target 4 → band 1 (relative); at 5.1% (>5) it is upper-breached even
// though the absolute deviation is only 1.1%p.
func TestCheckBandsAppliesFiveTwentyFiveRule(t *testing.T) {
	// total 1000 → 국내성장 50.5%, 해외성장 44.4%, 채권 5.1%
	summary, groups := bandFixture(t,
		map[string]float64{"국내성장": 35, "해외성장": 45, "채권": 4},
		map[string]string{"국내성장": "505", "해외성장": "444", "채권": "51"},
	)
	diags, err := checkBands(summary, groups)
	if err != nil {
		t.Fatalf("checkBands error: %v", err)
	}
	if len(diags) != 3 {
		t.Fatalf("want 3 diagnostics, got %d", len(diags))
	}
	byName := map[string]models.GroupDiagnostic{}
	for _, d := range diags {
		byName[d.RebalanceGroupName] = d
	}
	if got := byName["국내성장"].BandPct.Decimal; !got.Equal(decimal.NewFromInt(5)) {
		t.Errorf("국내성장 band: want 5 (absolute cap), got %s", got)
	}
	if got := byName["채권"].BandPct.Decimal; !got.Equal(decimal.NewFromInt(1)) {
		t.Errorf("채권 band: want 1 (25%% of 4), got %s", got)
	}
	if !byName["국내성장"].IsUpperBreached {
		t.Errorf("국내성장 at 50.5%% (upper 40) must be upper-breached")
	}
	if !byName["채권"].IsUpperBreached {
		t.Errorf("채권 at 5.1%% (upper 5) must be upper-breached")
	}
	if byName["해외성장"].IsUpperBreached || byName["해외성장"].IsLowerBreached {
		t.Errorf("해외성장 at 44.4%% (target 45, band 5) must be in-band")
	}
}

// TestCheckBandsGroupsComeFromInput: a group absent from holdings still gets a
// diagnostic (current 0%) and is lower-breached when its target exceeds the band.
func TestCheckBandsGroupsComeFromInput(t *testing.T) {
	summary, groups := bandFixture(t,
		map[string]float64{"국내성장": 50},
		map[string]string{"국내성장": "1000"},
	)
	groups = append(groups, models.Group{ID: uuidx.New(), Name: "금", TargetPercentage: 10})
	diags, err := checkBands(summary, groups)
	if err != nil {
		t.Fatalf("checkBands error: %v", err)
	}
	if len(diags) != 2 {
		t.Fatalf("want 2 diagnostics, got %d", len(diags))
	}
	for _, d := range diags {
		if d.RebalanceGroupName == "금" {
			if !d.CurrentPct.Decimal.IsZero() {
				t.Errorf("금 current: want 0, got %s", d.CurrentPct.Decimal)
			}
			if !d.IsLowerBreached {
				t.Errorf("금 at 0%% (lower 7.5) must be lower-breached")
			}
		}
	}
}

// TestCheckBandsMissingFXFails: a holding without KRW valuation must error —
// silently skipping it would understate the group and fire false alerts.
func TestCheckBandsMissingFXFails(t *testing.T) {
	summary, groups := bandFixture(t,
		map[string]float64{"해외성장": 25},
		map[string]string{"해외성장": "100"},
	)
	summary.Holdings[0].Holding.ValueKRW = nil
	if _, err := checkBands(summary, groups); err == nil {
		t.Fatal("want error for missing ValueKRW, got nil")
	}
}
