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
	got := sellReason(st, "국내배당", &bk)
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
	got := sellReason(st, "국내배당", &bk)
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
	got := preferredAccountTypesLabel("국내배당") // IRP & 연금 share top score 10
	if !strings.Contains(got, "IRP") || !strings.Contains(got, "연금저축") {
		t.Errorf("국내배당 preferred = %q, want IRP·연금저축", got)
	}
	if got := preferredAccountTypesLabel("해외배당"); got != "위탁" { // brokerage top score 8
		t.Errorf("해외배당 preferred = %q, want 위탁", got)
	}
}
