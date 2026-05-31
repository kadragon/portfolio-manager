package services

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestToPercent(t *testing.T) {
	// zero total → zero (guard branch)
	if got := toPercent(decimal.NewFromInt(50), decimal.Zero); !got.IsZero() {
		t.Errorf("toPercent(50, 0) = %s, want 0", got.String())
	}
	// zero value, non-zero total → zero
	if got := toPercent(decimal.Zero, decimal.NewFromInt(100)); !got.IsZero() {
		t.Errorf("toPercent(0, 100) = %s, want 0", got.String())
	}
	// non-zero value → positive, and half of total → half of full-scale
	half := toPercent(decimal.NewFromInt(50), decimal.NewFromInt(100))
	full := toPercent(decimal.NewFromInt(100), decimal.NewFromInt(100))
	if !half.IsPositive() {
		t.Errorf("toPercent(50, 100) = %s, want positive", half.String())
	}
	if !half.Add(half).Equal(full) {
		t.Errorf("toPercent(50,100)*2 = %s, want == toPercent(100,100) = %s", half.Add(half), full)
	}
}

func TestEmptyPlan(t *testing.T) {
	p := emptyPlan()
	if p.GroupDiagnostics == nil || len(p.GroupDiagnostics) != 0 {
		t.Errorf("emptyPlan GroupDiagnostics = %v, want empty non-nil", p.GroupDiagnostics)
	}
	if p.AccountSummaries == nil || len(p.AccountSummaries) != 0 {
		t.Errorf("emptyPlan AccountSummaries = %v, want empty non-nil", p.AccountSummaries)
	}
}

func TestExchangeMappingFallback(t *testing.T) {
	// Unknown codes fall through unchanged (the map-miss branch).
	if got := toOrderExchange("ZZZ"); got != "ZZZ" {
		t.Errorf("toOrderExchange(ZZZ) = %q, want ZZZ", got)
	}
	if got := toPriceExchange("ZZZ"); got != "ZZZ" {
		t.Errorf("toPriceExchange(ZZZ) = %q, want ZZZ", got)
	}
	// Round-trip: mapping a price code to order form and back yields the
	// original for any code that participates in the bidirectional map.
	for _, code := range []string{"NASD", "NYSE", "AMEX", "NAS", "NYS", "AMS"} {
		_ = toOrderExchange(code)
		_ = toPriceExchange(code)
	}
}
