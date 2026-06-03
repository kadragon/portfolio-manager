package services

import (
	"strings"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/shopspring/decimal"
)

func TestSellReasonTaxLocation(t *testing.T) {
	bk := models.AccountTypeBrokerage
	// target driven to 0 while uniform share was positive ⇒ pushed out by tax-location.
	st := accountGroupState{
		currentPct:           decimal.NewFromInt(20),
		targetPct:            decimal.Zero,
		upperPct:             decimal.NewFromInt(3),
		targetValueKRW:       decimal.Zero,
		mirrorTargetValueKRW: decimal.NewFromInt(1000),
	}
	available := map[string]bool{
		models.AccountTypeBrokerage: true,
		models.AccountTypeIRP:       true,
		models.AccountTypePension:   true,
	}
	got := sellReason(st, "국내배당", &bk, available)
	if !strings.Contains(got, "세금 위치 최적화") {
		t.Errorf("sell reason = %q, want tax-location explanation", got)
	}
	if !strings.Contains(got, "IRP") {
		t.Errorf("sell reason = %q, want preferred type IRP named", got)
	}
	if !strings.Contains(got, "위탁") {
		t.Errorf("sell reason = %q, want current account type 위탁 named", got)
	}
}

// TestSellReasonCapacityYield: 연금저축 sells down 해외성장 even though pension is a
// tax-home for it — because 국내배당 (higher pension score) won the capacity. The
// reason must NOT claim the group is tax-efficient elsewhere while naming this
// very account; it names the displacing group and notes the aggregate target is
// met in other accounts.
func TestSellReasonCapacityYield(t *testing.T) {
	pen := models.AccountTypePension
	available := map[string]bool{
		models.AccountTypeBrokerage: true,
		models.AccountTypePension:   true,
		models.AccountTypeISA:       true,
	}
	st := accountGroupState{
		currentPct:           decimal.NewFromFloat(23.44),
		targetPct:            decimal.NewFromFloat(2.25),
		upperPct:             decimal.NewFromInt(5),
		targetValueKRW:       decimal.NewFromInt(100),
		mirrorTargetValueKRW: decimal.NewFromInt(1000), // pushed out: 100 < 0.5*1000
	}
	got := sellReason(st, "해외성장", &pen, available)
	if !strings.Contains(got, "국내배당") {
		t.Errorf("sell reason = %q, want displacing group 국내배당 named", got)
	}
	if !strings.Contains(got, "다른 계좌") {
		t.Errorf("sell reason = %q, want aggregate-target note (다른 계좌)", got)
	}
	// must not produce the self-contradictory "X는 연금저축에서 효율 높아 연금저축 축소"
	if strings.Contains(got, "해외성장은(는) 연금저축에서 세금 효율이 높아 이 계좌(연금저축)") {
		t.Errorf("sell reason = %q, still self-contradictory", got)
	}
}

func TestSellReasonOverheated(t *testing.T) {
	bk := models.AccountTypeBrokerage
	// target == uniform share (no divergence); breach is genuine overweight.
	st := accountGroupState{
		currentPct:           decimal.NewFromInt(25),
		targetPct:            decimal.NewFromInt(15),
		upperPct:             decimal.NewFromInt(18),
		targetValueKRW:       decimal.NewFromInt(1000),
		mirrorTargetValueKRW: decimal.NewFromInt(1000),
	}
	got := sellReason(st, "국내배당", &bk, nil)
	if strings.Contains(got, "세금 위치 최적화") {
		t.Errorf("sell reason = %q, want non-tax (overheated) reason", got)
	}
	if !strings.Contains(got, "과열") {
		t.Errorf("sell reason = %q, want 과열 감축", got)
	}
}

func TestBuyReasonTaxLocation(t *testing.T) {
	pen := models.AccountTypePension
	// target far above uniform share ⇒ pulled in by tax-location.
	st := accountGroupState{
		targetPct:            decimal.NewFromInt(40),
		targetValueKRW:       decimal.NewFromInt(4000),
		mirrorTargetValueKRW: decimal.NewFromInt(1000),
	}
	got := buyReason(st, decimal.Zero, "국내배당", &pen)
	if !strings.Contains(got, "세금 위치 최적화") {
		t.Errorf("buy reason = %q, want tax-location explanation", got)
	}
	if !strings.Contains(got, "연금저축") {
		t.Errorf("buy reason = %q, want 연금저축 label", got)
	}
}

func TestBuyReasonShortfall(t *testing.T) {
	pen := models.AccountTypePension
	st := accountGroupState{
		targetPct:            decimal.NewFromInt(15),
		targetValueKRW:       decimal.NewFromInt(1000),
		mirrorTargetValueKRW: decimal.NewFromInt(1000),
	}
	got := buyReason(st, decimal.NewFromInt(10), "국내배당", &pen)
	if strings.Contains(got, "세금 위치 최적화") {
		t.Errorf("buy reason = %q, want shortfall reason", got)
	}
	if !strings.Contains(got, "부족분") {
		t.Errorf("buy reason = %q, want 부족분 보충", got)
	}
}

func TestPreferredAccountTypesLabel(t *testing.T) {
	got := preferredAccountTypesLabel("국내배당", nil) // IRP & 연금 share top score 10
	if !strings.Contains(got, "IRP") || !strings.Contains(got, "연금저축") {
		t.Errorf("국내배당 preferred = %q, want IRP·연금저축", got)
	}
	if got := preferredAccountTypesLabel("해외배당", nil); got != "위탁" { // brokerage top score 8
		t.Errorf("해외배당 preferred = %q, want 위탁", got)
	}
}

// TestPreferredAccountTypesLabelRestricted: the label must reflect only the
// account types the user actually holds, not the global tax preference. 국내배당
// globally prefers IRP·연금 (score 10), but a user with only {위탁, ISA} should
// see ISA (score 9 > 위탁 2), since that is where the group is concentrated.
func TestPreferredAccountTypesLabelRestricted(t *testing.T) {
	available := map[string]bool{
		models.AccountTypeBrokerage: true,
		models.AccountTypeISA:       true,
	}
	if got := preferredAccountTypesLabel("국내배당", available); got != "ISA" {
		t.Errorf("국내배당 preferred (위탁·ISA only) = %q, want ISA", got)
	}
}
