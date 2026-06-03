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

// TestComputeGroupNetActions: only aggregate band breaches produce trade needs —
// over-band sells down to target, under-band buys up to target, in-band nothing.
func TestComputeGroupNetActions(t *testing.T) {
	total := decimal.NewFromInt(1000)
	current := map[string]decimal.Decimal{
		"국내성장": decimal.NewFromInt(450), // 45% > 40 upper → sell 100 (to 350)
		"국내배당": decimal.NewFromInt(150), // 15% in band → nothing
		"해외성장": decimal.NewFromInt(250), // 25% in band → nothing
		"해외안정": decimal.NewFromInt(100), // 10% in band → nothing
		"해외배당": decimal.NewFromInt(50),  // 5% < 12 lower → buy 100 (to 150)
	}
	agg := buildGroupAggregates(current, standardTargets(), total)
	sellNeed, buyNeed := computeGroupNetActions(agg)

	if got := sellNeed["국내성장"]; !got.Equal(decimal.NewFromInt(100)) {
		t.Errorf("국내성장 sellNeed = %s, want 100", got.String())
	}
	if got := buyNeed["해외배당"]; !got.Equal(decimal.NewFromInt(100)) {
		t.Errorf("해외배당 buyNeed = %s, want 100", got.String())
	}
	for _, g := range []string{"국내배당", "해외성장", "해외안정"} {
		if sellNeed[g].IsPositive() || buyNeed[g].IsPositive() {
			t.Errorf("%s in band must produce no trade need; sell=%s buy=%s", g, sellNeed[g], buyNeed[g])
		}
	}
}

// TestAllocateSellsPrefersTaxAdvantaged: when an over-band group is held in both a
// taxable (위탁) and a tax-advantaged (연금) account, the sell is taken from the
// tax-advantaged account first to avoid realizing capital-gains tax.
func TestAllocateSellsPrefersTaxAdvantaged(t *testing.T) {
	bk := typedAcct(models.AccountTypeBrokerage)
	pen := typedAcct(models.AccountTypePension)
	positions := []accountPosition{
		{accountID: bk.ID, rebalanceGroup: "국내성장", ticker: "005930", valueKRW: decimal.NewFromInt(200)},
		{accountID: pen.ID, rebalanceGroup: "국내성장", ticker: "069500", valueKRW: decimal.NewFromInt(200)},
	}
	accountTypeByID := map[uuidx.UUID]*string{bk.ID: bk.AccountType, pen.ID: pen.AccountType}
	sellNeed := map[string]decimal.Decimal{"국내성장": decimal.NewFromInt(150)}

	s := NewRebalanceService()
	sell := s.allocateSells([]models.Account{bk, pen}, positions, sellNeed, accountTypeByID, false)

	if got := sell[[2]string{pen.ID.String(), "국내성장"}]; !got.Equal(decimal.NewFromInt(150)) {
		t.Errorf("pension sell = %s, want 150 (tax-advantaged exhausted first)", got)
	}
	if got := sell[[2]string{bk.ID.String(), "국내성장"}]; got.IsPositive() {
		t.Errorf("brokerage sell = %s, want 0 (avoid realizing tax)", got)
	}
}

// TestAllocateSellsLeastAppropriateFirst: among accounts of equal tax status, the
// group is sold first from where it is least tax-appropriate (lower placement
// score), nudging placement in the right direction.
func TestAllocateSellsLeastAppropriateFirst(t *testing.T) {
	// 해외성장 placement: pension 8 > ISA 7. Both tax-advantaged. Selling 해외성장
	// should drain ISA (score 7, less appropriate) before pension (score 8).
	isa := typedAcct(models.AccountTypeISA)
	pen := typedAcct(models.AccountTypePension)
	positions := []accountPosition{
		{accountID: isa.ID, rebalanceGroup: "해외성장", ticker: "133690", valueKRW: decimal.NewFromInt(200)},
		{accountID: pen.ID, rebalanceGroup: "해외성장", ticker: "368590", valueKRW: decimal.NewFromInt(200)},
	}
	accountTypeByID := map[uuidx.UUID]*string{isa.ID: isa.AccountType, pen.ID: pen.AccountType}
	sellNeed := map[string]decimal.Decimal{"해외성장": decimal.NewFromInt(150)}

	s := NewRebalanceService()
	sell := s.allocateSells([]models.Account{isa, pen}, positions, sellNeed, accountTypeByID, false)

	if got := sell[[2]string{isa.ID.String(), "해외성장"}]; !got.Equal(decimal.NewFromInt(150)) {
		t.Errorf("ISA sell = %s, want 150 (least appropriate drained first)", got)
	}
	if got := sell[[2]string{pen.ID.String(), "해외성장"}]; got.IsPositive() {
		t.Errorf("pension sell = %s, want 0 (more appropriate kept)", got)
	}
}
