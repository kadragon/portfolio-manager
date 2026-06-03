package services

import (
	"strings"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/shopspring/decimal"
)

func overAgg() groupAgg {
	return groupAgg{
		currentPct:      decimal.NewFromFloat(45.0),
		targetPct:       decimal.NewFromFloat(35.0),
		bandPct:         decimal.NewFromInt(5),
		isUpperBreached: true,
	}
}

func underAgg() groupAgg {
	return groupAgg{
		currentPct:      decimal.NewFromFloat(5.0),
		targetPct:       decimal.NewFromFloat(15.0),
		bandPct:         decimal.NewFromInt(3),
		isLowerBreached: true,
	}
}

// TestSellReasonAggregate: the sell reason is framed at the aggregate level (group
// over portfolio-wide band) and names the selling account only as the trade
// location — never implying a per-account target.
func TestSellReasonAggregate(t *testing.T) {
	pen := models.AccountTypePension
	got := sellReason(overAgg(), "국내성장", &pen, nil)
	if !strings.Contains(got, "합산 비중 초과") {
		t.Errorf("sell reason = %q, want aggregate over-band framing", got)
	}
	if !strings.Contains(got, "연금저축") {
		t.Errorf("sell reason = %q, want selling account 연금저축 named", got)
	}
	// 국내성장 tax-home is 위탁 (preferred label) — must appear, and must NOT use the
	// old self-contradictory per-account wording.
	if !strings.Contains(got, "위탁") {
		t.Errorf("sell reason = %q, want tax-preference 위탁 named", got)
	}
	if strings.Contains(got, "이 계좌 목표") {
		t.Errorf("sell reason = %q, must not imply a per-account target", got)
	}
}

// TestBuyReasonAggregate: the buy reason is framed at the aggregate level (group
// under portfolio-wide band), naming the buying account as the tax-preferred home.
func TestBuyReasonAggregate(t *testing.T) {
	pen := models.AccountTypePension
	got := buyReason(underAgg(), "국내배당", &pen, nil)
	if !strings.Contains(got, "합산 비중 부족") {
		t.Errorf("buy reason = %q, want aggregate under-band framing", got)
	}
	if !strings.Contains(got, "연금저축") {
		t.Errorf("buy reason = %q, want buying account 연금저축 named", got)
	}
}

func TestPreferredAccountTypesLabel(t *testing.T) {
	got := preferredAccountTypesLabel("국내배당", nil) // IRP & 연금 share top score 10
	if !strings.Contains(got, "IRP") || !strings.Contains(got, "연금저축") {
		t.Errorf("국내배당 preferred = %q, want IRP·연금저축", got)
	}
	if got := preferredAccountTypesLabel("해외배당", nil); got != "ISA" { // ISA top score 8 (corrected from brokerage)
		t.Errorf("해외배당 preferred = %q, want ISA", got)
	}
}

// TestPreferredAccountTypesLabelRestricted: the label reflects only the account
// types the user actually holds, not the global tax preference. 국내배당 globally
// prefers IRP·연금 (10), but a user with only {위탁, ISA} should see ISA (9 > 2).
func TestPreferredAccountTypesLabelRestricted(t *testing.T) {
	available := map[string]bool{
		models.AccountTypeBrokerage: true,
		models.AccountTypeISA:       true,
	}
	if got := preferredAccountTypesLabel("국내배당", available); got != "ISA" {
		t.Errorf("국내배당 preferred (위탁·ISA only) = %q, want ISA", got)
	}
}
