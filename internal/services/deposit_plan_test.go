package services_test

import (
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/services"
	"github.com/shopspring/decimal"
)

// TestBuildDepositPlanAllocatesDepositToUnderTargetGroups: simulating a cash
// deposit routes the new money to under-target groups in the deposited account
// (cash-flow rebalancing) without mutating the caller's inputs.
func TestBuildDepositPlanAllocatesDepositToUnderTargetGroups(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	// Stock total 940; deposit 60 → total 1000. After the deposit 해외안정 is
	// 90/1000 (target 10% → +10) and 해외배당 100/1000 (target 15% → +50);
	// no group breaches its upper band → zero sells.
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("350"),
		"국내배당": mustN("150"),
		"해외성장": mustN("250"),
		"해외안정": mustN("90"),
		"해외배당": mustN("100"),
	})
	a := makeAccount("A", "0")
	b := makeAccount("B", "0")
	holdingsByAccount := makeHoldingsByAccount([]models.Account{a, b}, stocks, map[string]map[string]string{
		"A": {"국내성장": "350", "국내배당": "150", "해외성장": "250", "해외안정": "90", "해외배당": "100"},
	})

	svc := services.NewRebalanceService()
	params := services.BuildPlanParams{
		Summary:           summary,
		Accounts:          []models.Account{a, b},
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	}
	plan, err := svc.BuildDepositPlan(params, a.ID, decimal.NewFromInt(60))
	if err != nil {
		t.Fatalf("BuildDepositPlan error: %v", err)
	}

	if len(plan.SellRecs) != 0 {
		t.Errorf("want 0 sells, got %d: %+v", len(plan.SellRecs), plan.SellRecs)
	}
	totalBuy := decimal.Zero
	buyByGroup := map[string]decimal.Decimal{}
	for _, r := range plan.BuyRecs {
		if r.AccountID != a.ID {
			t.Errorf("buy landed in account %s, want deposited account only", r.AccountName)
		}
		totalBuy = totalBuy.Add(r.AmountKRW.Decimal)
		buyByGroup[r.RebalanceGroupName] = buyByGroup[r.RebalanceGroupName].Add(r.AmountKRW.Decimal)
	}
	if !totalBuy.Equal(decimal.NewFromInt(60)) {
		t.Errorf("total buy = %s, want 60 (full deposit deployed)", totalBuy)
	}
	if !buyByGroup["해외안정"].Equal(decimal.NewFromInt(10)) {
		t.Errorf("해외안정 buy = %s, want 10", buyByGroup["해외안정"])
	}
	if !buyByGroup["해외배당"].Equal(decimal.NewFromInt(50)) {
		t.Errorf("해외배당 buy = %s, want 50", buyByGroup["해외배당"])
	}

	// Inputs untouched: the simulation must not leak cash into caller state.
	if !params.Accounts[0].CashBalance.IsZero() {
		t.Errorf("caller account CashBalance mutated to %v", params.Accounts[0].CashBalance)
	}

	// Unknown account → error.
	if _, err := svc.BuildDepositPlan(params, makeAccount("X", "0").ID, decimal.NewFromInt(60)); err == nil {
		t.Error("unknown account must error")
	}
}
