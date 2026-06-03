package services

import (
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
	"github.com/shopspring/decimal"
)

func standardTargets() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{
		"국내성장": decimal.NewFromInt(35),
		"국내배당": decimal.NewFromInt(15),
		"해외성장": decimal.NewFromInt(25),
		"해외안정": decimal.NewFromInt(10),
		"해외배당": decimal.NewFromInt(15),
	}
}

func typedAcct(t string) models.Account {
	at := t
	return models.Account{ID: uuidx.New(), AccountType: &at}
}

func sumGroupAcrossAccounts(target map[[2]string]decimal.Decimal, accounts []models.Account, group string) decimal.Decimal {
	sum := decimal.Zero
	for _, a := range accounts {
		sum = sum.Add(target[[2]string{a.ID.String(), group}])
	}
	return sum
}

// TestPlannerNilAccountUniformNotZero is the critical safety case: an
// unclassified (nil) account must receive the uniform global target, never zero
// — zero targets would make the engine recommend liquidating the whole account.
func TestPlannerNilAccountUniformNotZero(t *testing.T) {
	s := NewRebalanceService()
	acc := models.Account{ID: uuidx.New()} // AccountType nil
	aum := map[uuidx.UUID]decimal.Decimal{acc.ID: decimal.NewFromInt(1000)}

	target := s.planTargetsByAccountGroup([]models.Account{acc}, aum, standardTargets())

	// 국내성장 35% of 1000 = 350, etc. — uniform, not zero.
	want := map[string]int64{"국내성장": 350, "국내배당": 150, "해외성장": 250, "해외안정": 100, "해외배당": 150}
	total := decimal.Zero
	for g, w := range want {
		got := target[[2]string{acc.ID.String(), g}]
		if !got.Equal(decimal.NewFromInt(w)) {
			t.Errorf("nil account %s target = %s, want %d", g, got.String(), w)
		}
		total = total.Add(got)
	}
	if !total.Equal(decimal.NewFromInt(1000)) {
		t.Errorf("nil account total target = %s, want 1000 (fully invested, no liquidation)", total.String())
	}
}

// TestPlannerSingleTypeMirrors confirms a homogeneous portfolio (all brokerage)
// gets the uniform mirror — per-account target = globalPct * accountAUM.
func TestPlannerSingleTypeMirrors(t *testing.T) {
	s := NewRebalanceService()
	a := typedAcct(models.AccountTypeBrokerage)
	b := typedAcct(models.AccountTypeBrokerage)
	aum := map[uuidx.UUID]decimal.Decimal{
		a.ID: decimal.NewFromInt(600),
		b.ID: decimal.NewFromInt(400),
	}
	target := s.planTargetsByAccountGroup([]models.Account{a, b}, aum, standardTargets())

	// 국내배당 15%: a=90, b=60 (proportional to AUM).
	if got := target[[2]string{a.ID.String(), "국내배당"}]; !got.Equal(decimal.NewFromInt(90)) {
		t.Errorf("a 국내배당 = %s, want 90", got.String())
	}
	if got := target[[2]string{b.ID.String(), "국내배당"}]; !got.Equal(decimal.NewFromInt(60)) {
		t.Errorf("b 국내배당 = %s, want 60", got.String())
	}
}

// TestPlannerTaxLocation is the core Phase 2 behavior: with a brokerage and an
// IRP account, 국내배당 concentrates in IRP (tax-deferred) and 해외배당 in
// brokerage (post-2025 the foreign-dividend shelter is gone).
func TestPlannerTaxLocation(t *testing.T) {
	s := NewRebalanceService()
	bk := typedAcct(models.AccountTypeBrokerage)
	irp := typedAcct(models.AccountTypeIRP)
	accounts := []models.Account{bk, irp}
	// Equal AUM so capacity isn't the deciding factor — preference is.
	aum := map[uuidx.UUID]decimal.Decimal{
		bk.ID:  decimal.NewFromInt(500),
		irp.ID: decimal.NewFromInt(500),
	}
	target := s.planTargetsByAccountGroup(accounts, aum, standardTargets())

	// 국내배당 (global 15% of 1000 = 150) should land entirely in IRP (capacity 500 >> 150).
	irpDom := target[[2]string{irp.ID.String(), "국내배당"}]
	bkDom := target[[2]string{bk.ID.String(), "국내배당"}]
	if !irpDom.GreaterThan(bkDom) {
		t.Errorf("국내배당 should prefer IRP: irp=%s brokerage=%s", irpDom.String(), bkDom.String())
	}
	if !bkDom.IsZero() {
		t.Errorf("국내배당 should fully fit in IRP, brokerage share = %s, want 0", bkDom.String())
	}

	// 해외배당 (global 15% = 150) should prefer brokerage.
	bkFor := target[[2]string{bk.ID.String(), "해외배당"}]
	irpFor := target[[2]string{irp.ID.String(), "해외배당"}]
	if !bkFor.GreaterThan(irpFor) {
		t.Errorf("해외배당 should prefer brokerage: brokerage=%s irp=%s", bkFor.String(), irpFor.String())
	}

	// Conservation: each group's total across accounts == global target value.
	for g, pct := range standardTargets() {
		want := pct.Div(decimal.NewFromInt(100)).Mul(decimal.NewFromInt(1000))
		got := sumGroupAcrossAccounts(target, accounts, g)
		if !got.Equal(want) {
			t.Errorf("group %s total = %s, want %s (conservation)", g, got.String(), want.String())
		}
	}

	// Each account fully invested (targets sum to its AUM).
	for _, a := range accounts {
		sum := decimal.Zero
		for _, g := range _groupOrder {
			sum = sum.Add(target[[2]string{a.ID.String(), g}])
		}
		if !sum.Equal(decimal.NewFromInt(500)) {
			t.Errorf("account %s targets sum = %s, want 500", a.ID.String(), sum.String())
		}
	}
}

// TestPlannerEmptyAUM returns empty without panicking.
func TestPlannerEmptyAUM(t *testing.T) {
	s := NewRebalanceService()
	acc := typedAcct(models.AccountTypeBrokerage)
	aum := map[uuidx.UUID]decimal.Decimal{acc.ID: decimal.Zero}
	target := s.planTargetsByAccountGroup([]models.Account{acc}, aum, standardTargets())
	if len(target) != 0 {
		t.Errorf("zero-AUM portfolio should yield empty targets, got %d entries", len(target))
	}
}
