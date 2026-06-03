package services_test

import (
	"testing"
	"time"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/services"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// --- helpers ---

func makeGroup(name string, targetPct float64) models.Group {
	return models.Group{ID: uuidx.New(), Name: name, TargetPercentage: targetPct, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func makeStock(ticker string, groupID uuidx.UUID) models.Stock {
	return models.Stock{ID: uuidx.New(), Ticker: ticker, GroupID: groupID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

// makeAccount builds a brokerage-type account. The pre-eligibility engine
// treated every account as "anything allowed"; brokerage preserves that for the
// existing math-focused tests. Use makeTypedAccount to exercise eligibility.
func makeAccount(name string, cashBalance string) models.Account {
	a := makeTypedAccount(name, cashBalance, models.AccountTypeBrokerage)
	return a
}

func makeTypedAccount(name, cashBalance, accountType string) models.Account {
	cb, _ := numeric.FromString(cashBalance)
	at := accountType
	return models.Account{ID: uuidx.New(), Name: name, CashBalance: cb, AccountType: &at, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func makeHolding(accountID, stockID uuidx.UUID, quantity string) models.Holding {
	qty, _ := numeric.FromString(quantity)
	return models.Holding{ID: uuidx.New(), AccountID: accountID, StockID: stockID, Quantity: qty, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func makeStandardGroups() []models.Group {
	return []models.Group{
		makeGroup("국내성장", 35.0),
		makeGroup("국내배당", 15.0),
		makeGroup("해외성장", 25.0),
		makeGroup("해외안정", 10.0),
		makeGroup("해외배당", 15.0),
	}
}

func makeStandardStocks(groups []models.Group) map[string]models.Stock {
	byName := map[string]models.Group{}
	for _, g := range groups {
		byName[g.Name] = g
	}
	return map[string]models.Stock{
		"국내성장": makeStock("005930", byName["국내성장"].ID),
		"국내배당": makeStock("000660", byName["국내배당"].ID),
		"해외성장": makeStock("QQQ", byName["해외성장"].ID),
		"해외안정": makeStock("VOO", byName["해외안정"].ID),
		"해외배당": makeStock("SCHD", byName["해외배당"].ID),
	}
}

// makeSummary builds a PortfolioSummary with price=1, value_krw=value for each sleeve.
func makeSummary(groups []models.Group, stocks map[string]models.Stock, sleeveValues map[string]numeric.Decimal) models.PortfolioSummary {
	byName := map[string]models.Group{}
	for _, g := range groups {
		byName[g.Name] = g
	}

	pairs := []models.GroupHoldingPair{}
	totalValue := numeric.Zero
	for sleeveName, value := range sleeveValues {
		stock := stocks[sleeveName]
		currency := "KRW"
		if sleeveName == "해외성장" || sleeveName == "해외안정" || sleeveName == "해외배당" {
			currency = "USD"
		}
		vKRW := value
		pairs = append(pairs, models.GroupHoldingPair{
			Group: byName[sleeveName],
			Holding: models.StockHoldingWithPrice{
				Stock:    stock,
				Quantity: value,
				Price:    mustN("1"),
				Currency: currency,
				Name:     stock.Ticker,
				ValueKRW: &vKRW,
			},
		})
		totalValue = numeric.Wrap(totalValue.Add(value.Decimal))
	}

	return models.PortfolioSummary{
		Holdings:    pairs,
		TotalValue:  totalValue,
		TotalAssets: totalValue,
	}
}

func makeHoldingsByAccount(
	accounts []models.Account,
	stocks map[string]models.Stock,
	perAccountSleeveValues map[string]map[string]string,
) map[uuidx.UUID][]models.Holding {
	byName := map[string]models.Account{}
	for _, a := range accounts {
		byName[a.Name] = a
	}
	result := map[uuidx.UUID][]models.Holding{}
	for accName, sleeveValues := range perAccountSleeveValues {
		acc := byName[accName]
		var holdings []models.Holding
		for sleeveName, qtyStr := range sleeveValues {
			qty, _ := numeric.FromString(qtyStr)
			if qty.IsZero() {
				continue
			}
			holdings = append(holdings, makeHolding(acc.ID, stocks[sleeveName].ID, qtyStr))
		}
		result[acc.ID] = holdings
	}
	return result
}

// --- tests ---

// TestBuildPlanNoTradesWhenAggregateInBand is the thesis of the aggregate-band
// redesign: when every group (and the region) is within its aggregate band, the
// engine emits ZERO trades — even if the per-account placement is tax-suboptimal
// (위탁 holding 국내배당, 연금 holding 국내성장). Rebalancing rides only band
// breaches; tax-location is corrected gradually on the trades those breaches
// force, never by proactive relocation (which would realize capital-gains tax).
func TestBuildPlanNoTradesWhenAggregateInBand(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	// On-target aggregate: 35/15/25/10/15, region KR=50 / US=50 — all in band.
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("350"),
		"국내배당": mustN("150"),
		"해외성장": mustN("250"),
		"해외안정": mustN("100"),
		"해외배당": mustN("150"),
	})
	brokerage := makeTypedAccount("위탁", "0", models.AccountTypeBrokerage)
	pension := makeTypedAccount("연금", "0", models.AccountTypePension)
	// Deliberately tax-suboptimal split (위탁 holds 국내배당; 연금 holds 국내성장),
	// but the aggregate is exactly on target.
	holdingsByAccount := map[uuidx.UUID][]models.Holding{
		brokerage.ID: {
			makeHolding(brokerage.ID, stocks["국내배당"].ID, "150"),
			makeHolding(brokerage.ID, stocks["해외성장"].ID, "250"),
			makeHolding(brokerage.ID, stocks["해외안정"].ID, "100"),
			makeHolding(brokerage.ID, stocks["해외배당"].ID, "150"),
		},
		pension.ID: {
			makeHolding(pension.ID, stocks["국내성장"].ID, "350"),
		},
	}

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          []models.Account{brokerage, pension},
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	if len(plan.SellRecs) != 0 {
		t.Errorf("in-band aggregate must produce 0 sells, got %d: %+v", len(plan.SellRecs), plan.SellRecs)
	}
	if len(plan.BuyRecs) != 0 {
		t.Errorf("in-band aggregate must produce 0 buys, got %d: %+v", len(plan.BuyRecs), plan.BuyRecs)
	}
}

func TestBuildPlanFlagsUpperAndLowerBandBreaches(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("500"),
		"국내배당": mustN("120"),
		"해외성장": mustN("240"),
		"해외안정": mustN("100"),
		"해외배당": mustN("40"),
	})
	account := makeAccount("A", "0")
	holdingsByAccount := makeHoldingsByAccount([]models.Account{account}, stocks, map[string]map[string]string{
		"A": {"국내성장": "500", "국내배당": "120", "해외성장": "240", "해외안정": "100", "해외배당": "40"},
	})

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          []models.Account{account},
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	diagByName := map[string]models.GroupDiagnostic{}
	for _, d := range plan.GroupDiagnostics {
		diagByName[d.RebalanceGroupName] = d
	}

	if !diagByName["국내성장"].IsUpperBreached {
		t.Error("국내성장: expected upper breached")
	}
	if !diagByName["해외배당"].IsLowerBreached {
		t.Error("해외배당: expected lower breached")
	}
}

func TestBuildPlanRegionTrigger(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("400"),
		"국내배당": mustN("200"),
		"해외성장": mustN("180"),
		"해외안정": mustN("120"),
		"해외배당": mustN("100"),
	})
	account := makeAccount("A", "0")
	holdingsByAccount := makeHoldingsByAccount([]models.Account{account}, stocks, map[string]map[string]string{
		"A": {"국내성장": "400", "국내배당": "200", "해외성장": "180", "해외안정": "120", "해외배당": "100"},
	})

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          []models.Account{account},
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	rd := plan.RegionDiagnostic
	if !numericEq(rd.TargetKRPct, "50") {
		t.Errorf("TargetKRPct: got %v, want 50", rd.TargetKRPct)
	}
	if !numericEq(rd.CurrentKRPct, "60") {
		t.Errorf("CurrentKRPct: got %v, want 60", rd.CurrentKRPct)
	}
	if !rd.IsTriggered {
		t.Error("region diagnostic should be triggered")
	}
}

// TestBuildPlanSellsOverBandGroupToTarget: an over-band group is sold down to its
// TARGET (not a half-rule midpoint). With both holders taxable (brokerage) and
// equal placement score, the larger holding is drained first.
func TestBuildPlanSellsOverBandGroupToTarget(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	// total 990; 국내성장 500 = 50.5% (> upper 40) → sell to target 35% (346.5) = 153.5.
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("500"),
		"국내배당": mustN("100"),
		"해외성장": mustN("200"),
		"해외안정": mustN("90"),
		"해외배당": mustN("100"),
	})
	accounts := []models.Account{makeAccount("A", "0"), makeAccount("B", "0")}
	holdingsByAccount := makeHoldingsByAccount(accounts, stocks, map[string]map[string]string{
		"A": {"국내성장": "300", "국내배당": "100", "해외성장": "200", "해외안정": "90", "해외배당": "100"},
		"B": {"국내성장": "200"},
	})

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          accounts,
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	totalGrowthSell := numeric.Zero
	byAccount := map[string]numeric.Decimal{"A": numeric.Zero, "B": numeric.Zero}
	for _, rec := range plan.SellRecs {
		if rec.RebalanceGroupName == "국내성장" {
			totalGrowthSell = numeric.Wrap(totalGrowthSell.Add(rec.AmountKRW.Decimal))
			byAccount[rec.AccountName] = numeric.Wrap(byAccount[rec.AccountName].Add(rec.AmountKRW.Decimal))
		}
	}
	if !numericEq(totalGrowthSell, "153.5") {
		t.Errorf("국내성장 total sell = %v, want 153.5 (sell to target)", totalGrowthSell)
	}
	// A holds 300, B holds 200 — both brokerage; larger holder (A) drained first,
	// and 153.5 < 300 so B is untouched.
	if !numericEq(byAccount["A"], "153.5") {
		t.Errorf("A 국내성장 sell = %v, want 153.5", byAccount["A"])
	}
	if !byAccount["B"].IsZero() {
		t.Errorf("B 국내성장 sell = %v, want 0", byAccount["B"])
	}
}

func TestBuildPlanSellAllocationFixedDenominator(t *testing.T) {
	groups := makeStandardGroups()
	byName := map[string]models.Group{}
	for _, g := range groups {
		byName[g.Name] = g
	}

	growthA := makeStock("100001", byName["국내성장"].ID)
	growthB := makeStock("100002", byName["국내성장"].ID)
	growthC := makeStock("100003", byName["국내성장"].ID)
	krDiv := makeStock("200001", byName["국내배당"].ID)
	usGrowth := makeStock("QQQ", byName["해외성장"].ID)
	usStable := makeStock("VOO", byName["해외안정"].ID)
	usDiv := makeStock("SCHD", byName["해외배당"].ID)

	v250, v150, v100, v200 := mustN("250"), mustN("150"), mustN("100"), mustN("200")

	summary := models.PortfolioSummary{
		Holdings: []models.GroupHoldingPair{
			{Group: byName["국내성장"], Holding: models.StockHoldingWithPrice{Stock: growthA, Quantity: v250, Price: mustN("1"), Currency: "KRW", Name: "100001", ValueKRW: &v250}},
			{Group: byName["국내성장"], Holding: models.StockHoldingWithPrice{Stock: growthB, Quantity: v150, Price: mustN("1"), Currency: "KRW", Name: "100002", ValueKRW: &v150}},
			{Group: byName["국내성장"], Holding: models.StockHoldingWithPrice{Stock: growthC, Quantity: v100, Price: mustN("1"), Currency: "KRW", Name: "100003", ValueKRW: &v100}},
			{Group: byName["국내배당"], Holding: models.StockHoldingWithPrice{Stock: krDiv, Quantity: v100, Price: mustN("1"), Currency: "KRW", Name: "200001", ValueKRW: &v100}},
			{Group: byName["해외성장"], Holding: models.StockHoldingWithPrice{Stock: usGrowth, Quantity: v200, Price: mustN("1"), Currency: "USD", Name: "QQQ", ValueKRW: &v200}},
			{Group: byName["해외안정"], Holding: models.StockHoldingWithPrice{Stock: usStable, Quantity: v100, Price: mustN("1"), Currency: "USD", Name: "VOO", ValueKRW: &v100}},
			{Group: byName["해외배당"], Holding: models.StockHoldingWithPrice{Stock: usDiv, Quantity: v100, Price: mustN("1"), Currency: "USD", Name: "SCHD", ValueKRW: &v100}},
		},
		TotalValue:  mustN("1000"),
		TotalAssets: mustN("1000"),
	}

	account := makeAccount("A", "0")
	holdingsByAccount := map[uuidx.UUID][]models.Holding{
		account.ID: {
			makeHolding(account.ID, growthA.ID, "250"),
			makeHolding(account.ID, growthB.ID, "150"),
			makeHolding(account.ID, growthC.ID, "100"),
			makeHolding(account.ID, krDiv.ID, "100"),
			makeHolding(account.ID, usGrowth.ID, "200"),
			makeHolding(account.ID, usStable.ID, "100"),
			makeHolding(account.ID, usDiv.ID, "100"),
		},
	}

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          []models.Account{account},
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            []models.Stock{growthA, growthB, growthC, krDiv, usGrowth, usStable, usDiv},
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	growthSells := []models.RebalanceRecommendation{}
	for _, rec := range plan.SellRecs {
		if rec.RebalanceGroupName == "국내성장" && rec.AccountName == "A" {
			growthSells = append(growthSells, rec)
		}
	}
	if len(growthSells) != 3 {
		t.Errorf("want 3 growth sells, got %d", len(growthSells))
	}
	byTicker := map[string]numeric.Decimal{}
	for _, rec := range growthSells {
		byTicker[rec.Ticker] = rec.AmountKRW
	}
	// 국내성장 500 = 50% over → sell to target 35% (350) = 150, split across the
	// three holdings by fixed denominator 250:150:100 → 75:45:30.
	if !numericEq(byTicker["100001"], "75") {
		t.Errorf("100001: want 75, got %v", byTicker["100001"])
	}
	if !numericEq(byTicker["100002"], "45") {
		t.Errorf("100002: want 45, got %v", byTicker["100002"])
	}
	if !numericEq(byTicker["100003"], "30") {
		t.Errorf("100003: want 30, got %v", byTicker["100003"])
	}
}

func TestBuildPlanSkipsSellWhenOnlyLowerBreachExists(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("380"),
		"국내배당": mustN("160"),
		"해외성장": mustN("260"),
		"해외안정": mustN("100"),
		"해외배당": mustN("100"),
	})
	account := makeAccount("A", "50")
	holdingsByAccount := makeHoldingsByAccount([]models.Account{account}, stocks, map[string]map[string]string{
		"A": {"국내성장": "380", "국내배당": "160", "해외성장": "260", "해외안정": "100", "해외배당": "100"},
	})

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          []models.Account{account},
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	if len(plan.SellRecs) != 0 {
		t.Errorf("want 0 sells, got %d", len(plan.SellRecs))
	}
	if len(plan.BuyRecs) != 1 {
		t.Errorf("want 1 buy, got %d", len(plan.BuyRecs))
	}
	buy := plan.BuyRecs[0]
	if buy.RebalanceGroupName != "해외배당" {
		t.Errorf("buy group: want 해외배당, got %q", buy.RebalanceGroupName)
	}
	if buy.Ticker != "SCHD" {
		t.Errorf("buy ticker: want SCHD, got %q", buy.Ticker)
	}
	if !numericEq(buy.AmountKRW, "50") {
		t.Errorf("buy amount_krw: want 50, got %v", buy.AmountKRW)
	}
}

func TestBuildPlanReinvestsCashWithinSameAccount(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("500"),
		"해외성장": mustN("20"),
	})
	accounts := []models.Account{makeAccount("A", "0"), makeAccount("B", "0")}
	holdingsByAccount := makeHoldingsByAccount(accounts, stocks, map[string]map[string]string{
		"A": {"국내성장": "500", "해외성장": "20"},
		"B": {},
	})

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          accounts,
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	if len(plan.SellRecs) == 0 {
		t.Error("expected some sells")
	}
	for _, rec := range plan.SellRecs {
		if rec.AccountName != "A" {
			t.Errorf("sell should be from A, got %q", rec.AccountName)
		}
	}
	if len(plan.BuyRecs) == 0 {
		t.Error("expected some buys")
	}
	for _, rec := range plan.BuyRecs {
		if rec.AccountName != "A" {
			t.Errorf("buy should be for A, got %q", rec.AccountName)
		}
	}
}

// TestBuildPlanSellsFromLargerHolderWhenSameTaxStatus: 국내성장 over-band, held in
// two brokerage accounts (same tax status, same placement score). The larger
// holding is drained first and the sell stays within it; the smaller holder is
// untouched.
func TestBuildPlanSellsFromLargerHolderWhenSameTaxStatus(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	// total 1200; 국내성장 550 = 45.8% (> 40) → sell to 35% (420) = 130 (< A's 350).
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("550"),
		"국내배당": mustN("150"),
		"해외성장": mustN("250"),
		"해외안정": mustN("100"),
		"해외배당": mustN("150"),
	})
	accounts := []models.Account{makeAccount("A", "0"), makeAccount("B", "0")}
	holdingsByAccount := makeHoldingsByAccount(accounts, stocks, map[string]map[string]string{
		"A": {"국내성장": "350", "국내배당": "150", "해외성장": "250", "해외안정": "100", "해외배당": "150"},
		"B": {"국내성장": "200"},
	})

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          accounts,
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	if len(plan.SellRecs) == 0 {
		t.Error("expected sells")
	}
	for _, rec := range plan.SellRecs {
		if rec.AccountName != "A" {
			t.Errorf("sell should be from larger holder A, got %q", rec.AccountName)
		}
	}
	// Cash isolation: B never sold, so B never buys.
	for _, rec := range plan.BuyRecs {
		if rec.AccountName == "B" {
			t.Errorf("B had no sell cash and should not buy, got buy rec %+v", rec)
		}
	}
}

func TestBuildPlanUnmetGroupProducesWarning(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	account := makeAccount("A", "100")
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("800"),
		"국내배당": mustN("100"),
	})
	holdingsByAccount := makeHoldingsByAccount([]models.Account{account}, stocks, map[string]map[string]string{
		"A": {"국내성장": "800", "국내배당": "100"},
	})

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          []models.Account{account},
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	for _, rec := range plan.BuyRecs {
		switch rec.RebalanceGroupName {
		case "해외성장", "해외안정", "해외배당":
			t.Errorf("unexpected foreign buy rec: %+v", rec)
		}
	}

	if len(plan.AccountSummaries) != 1 {
		t.Fatalf("want 1 account summary, got %d", len(plan.AccountSummaries))
	}
	sumA := plan.AccountSummaries[0]
	if !containsString(sumA.UnmetGroups, "해외성장") {
		t.Errorf("expected 해외성장 in unmet groups, got %v", sumA.UnmetGroups)
	}
	if !containsString(sumA.UnmetGroups, "해외배당") {
		t.Errorf("expected 해외배당 in unmet groups, got %v", sumA.UnmetGroups)
	}
	if sumA.UnusedCashKRW.IsZero() {
		t.Error("expected unused_cash_krw > 0")
	}
}

func TestBuildPlanCashIsolationInvariant(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	accountA := makeAccount("A", "0")
	accountB := makeAccount("B", "0")
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("500"),
		"해외성장": mustN("100"),
	})
	holdingsByAccount := map[uuidx.UUID][]models.Holding{
		accountA.ID: {makeHolding(accountA.ID, stocks["국내성장"].ID, "500")},
		accountB.ID: {makeHolding(accountB.ID, stocks["해외성장"].ID, "100")},
	}

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          []models.Account{accountA, accountB},
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	sumA := findAccountSummary(plan, accountA.ID)
	sumB := findAccountSummary(plan, accountB.ID)

	aTotalBuy := numeric.Zero
	for _, r := range sumA.BuyRecs {
		aTotalBuy = numeric.Wrap(aTotalBuy.Add(r.AmountKRW.Decimal))
	}
	aLimit := numeric.Wrap(accountA.CashBalance.Add(sumA.SellCashKRW.Decimal))
	if aTotalBuy.GreaterThan(aLimit.Decimal) {
		t.Errorf("A total buy %v exceeds A cash+sell %v", aTotalBuy, aLimit)
	}

	bTotalBuy := numeric.Zero
	for _, r := range sumB.BuyRecs {
		bTotalBuy = numeric.Wrap(bTotalBuy.Add(r.AmountKRW.Decimal))
	}
	bLimit := numeric.Wrap(accountB.CashBalance.Add(sumB.SellCashKRW.Decimal))
	if bTotalBuy.GreaterThan(bLimit.Decimal) {
		t.Errorf("B total buy %v exceeds B cash+sell %v", bTotalBuy, bLimit)
	}
}

func TestBuildPlanSameNameAccountsCorrectAttribution(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	accountA := makeAccount("계좌", "0")
	accountB := makeAccount("계좌", "0")
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("600"),
	})
	holdingsByAccount := map[uuidx.UUID][]models.Holding{
		accountA.ID: {makeHolding(accountA.ID, stocks["국내성장"].ID, "600")},
		accountB.ID: {},
	}

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          []models.Account{accountA, accountB},
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	if len(plan.AccountSummaries) != 2 {
		t.Fatalf("want 2 account summaries, got %d", len(plan.AccountSummaries))
	}
	sA := findAccountSummary(plan, accountA.ID)
	sB := findAccountSummary(plan, accountB.ID)

	if len(sA.SellRecs) == 0 {
		t.Error("account A should have sell recs")
	}
	for _, r := range sA.SellRecs {
		if r.Ticker != "005930" {
			t.Errorf("A sell ticker: want 005930, got %q", r.Ticker)
		}
	}
	if len(sB.SellRecs) != 0 {
		t.Errorf("account B should have no sell recs, got %d", len(sB.SellRecs))
	}
}

// TestBuildPlanPortfolioFallbackBuysIntoNewGroup: 해외성장 is under-band and the
// only 해외성장 security (QQQ) is held in a different account (B). Account A, which
// sold its over-band 국내성장, buys QQQ for the under-band group via the
// portfolio-wide candidate fallback (it holds no 해외성장 of its own).
func TestBuildPlanPortfolioFallbackBuysIntoNewGroup(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	accountA := makeAccount("A", "10000")
	accountB := makeAccount("B", "0")
	// total 32000; 해외성장 2000 = 6.25% (< 20) under → buy ~6000; 국내성장 30000 over.
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("30000"),
		"해외성장": mustN("2000"),
	})
	holdingsByAccount := map[uuidx.UUID][]models.Holding{
		accountA.ID: {makeHolding(accountA.ID, stocks["국내성장"].ID, "30000")},
		accountB.ID: {makeHolding(accountB.ID, stocks["해외성장"].ID, "2000")},
	}

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          []models.Account{accountA, accountB},
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	found := false
	for _, rec := range plan.BuyRecs {
		if rec.AccountName == "A" && rec.Ticker == "QQQ" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected A to buy QQQ (해외성장) via portfolio fallback; buys=%+v", plan.BuyRecs)
	}
}

// TestBuildPlanEtfClassificationUnblocksPensionBuy guards the data→eligibility
// chain: a 연금저축 account under-band in 국내배당 may buy the domestic dividend ETF
// only once it is classified (asset_class=etf). While unclassified (isETF=false),
// canHold blocks the buy and 국내배당 surfaces as unmet.
func TestBuildPlanEtfClassificationUnblocksPensionBuy(t *testing.T) {
	run := func(t *testing.T, assetClass *string) models.AccountRebalanceSummary {
		t.Helper()
		groups := makeStandardGroups()
		stocks := makeStandardStocks(groups)
		dom := stocks["국내배당"] // ticker 000660 (domestic)
		dom.AssetClass = assetClass
		stocks["국내배당"] = dom

		// total 150; 국내배당 10 = 6.7% (< 12) under → buy; 국내성장 140 over → sell.
		pension := makeTypedAccount("연금", "8500", models.AccountTypePension)
		summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
			"국내성장": mustN("140"),
			"국내배당": mustN("10"),
		})
		holdingsByAccount := makeHoldingsByAccount([]models.Account{pension}, stocks, map[string]map[string]string{
			"연금": {"국내성장": "140", "국내배당": "10"},
		})

		plan, err := services.NewRebalanceService().BuildPlan(services.BuildPlanParams{
			Summary:           summary,
			Accounts:          []models.Account{pension},
			HoldingsByAccount: holdingsByAccount,
			Groups:            groups,
			Stocks:            stockSlice(stocks),
		})
		if err != nil {
			t.Fatalf("BuildPlan error: %v", err)
		}
		for _, s := range plan.AccountSummaries {
			if s.AccountID == pension.ID {
				return s
			}
		}
		t.Fatal("pension summary missing")
		return models.AccountRebalanceSummary{}
	}

	etf := "etf"
	classified := run(t, &etf)
	boughtDividendETF := false
	for _, rec := range classified.BuyRecs {
		if rec.Ticker == "000660" {
			boughtDividendETF = true
		}
	}
	if !boughtDividendETF {
		t.Errorf("classified=etf: expected 연금 to buy 000660, BuyRecs=%v", classified.BuyRecs)
	}
	if containsString(classified.UnmetGroups, "국내배당") {
		t.Errorf("classified=etf: 국내배당 should not be unmet, got %v", classified.UnmetGroups)
	}

	unclassified := run(t, nil)
	for _, rec := range unclassified.BuyRecs {
		if rec.Ticker == "000660" {
			t.Errorf("unclassified: 연금 must NOT buy 000660 (isETF=false blocks IRP/연금)")
		}
	}
	if !containsString(unclassified.UnmetGroups, "국내배당") {
		t.Errorf("unclassified: 국내배당 should be unmet, got %v", unclassified.UnmetGroups)
	}
}

func TestBuildPlanAccountSummariesPopulated(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	accounts := []models.Account{makeAccount("Alpha", "0"), makeAccount("Beta", "0")}
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("300"),
		"해외성장": mustN("200"),
	})
	holdingsByAccount := makeHoldingsByAccount(accounts, stocks, map[string]map[string]string{
		"Alpha": {"국내성장": "300"},
		"Beta":  {"해외성장": "200"},
	})

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          accounts,
		HoldingsByAccount: holdingsByAccount,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	if len(plan.AccountSummaries) != 2 {
		t.Fatalf("want 2 account summaries, got %d", len(plan.AccountSummaries))
	}
	names := map[string]bool{}
	for _, s := range plan.AccountSummaries {
		names[s.AccountName] = true
	}
	if !names["Alpha"] || !names["Beta"] {
		t.Errorf("expected Alpha and Beta, got %v", names)
	}
}

// --- test helpers ---

// TestBuildPlanNilAccountNotLiquidated guards the Phase 2 safety net: an
// unclassified (nil account_type) account holding a balanced, on-target mix must
// NOT be told to sell everything. The planner routes nil accounts to the uniform
// global target, so a balanced portfolio stays put.
func TestBuildPlanNilAccountNotLiquidated(t *testing.T) {
	groups := makeStandardGroups()
	stocks := makeStandardStocks(groups)
	// On-target sleeve values (35/15/25/10/15).
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("350"),
		"국내배당": mustN("150"),
		"해외성장": mustN("250"),
		"해외안정": mustN("100"),
		"해외배당": mustN("150"),
	})
	// Account with NO account_type set (nil).
	acc := models.Account{ID: uuidx.New(), Name: "미분류", CashBalance: numeric.Zero, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	holdings := makeHoldingsByAccount([]models.Account{acc}, stocks, map[string]map[string]string{
		"미분류": {"국내성장": "350", "국내배당": "150", "해외성장": "250", "해외안정": "100", "해외배당": "150"},
	})

	svc := services.NewRebalanceService()
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:           summary,
		Accounts:          []models.Account{acc},
		HoldingsByAccount: holdings,
		Groups:            groups,
		Stocks:            stockSlice(stocks),
	})
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if len(plan.SellRecs) != 0 {
		t.Fatalf("nil-type balanced account must not be liquidated; got %d sell recs: %+v", len(plan.SellRecs), plan.SellRecs)
	}
}

// makeStockETF builds a stock pre-classified as an ETF.
func makeStockETF(ticker string, groupID uuidx.UUID) models.Stock {
	s := makeStock(ticker, groupID)
	ac := "etf"
	s.AssetClass = &ac
	return s
}

// TestBuildPlanBlocksIneligibleBuyInIRP proves the eligibility guard: an IRP
// account underweight in 해외배당 whose only 해외배당 candidate is the US-listed
// ETF SCHD must NOT receive a SCHD buy (IRP cannot hold foreign-listed
// securities) and must report 해외배당 as unmet — even though a SCHD snapshot
// exists. A brokerage account in the same scenario WOULD buy SCHD.
func TestBuildPlanBlocksIneligibleBuyInIRP(t *testing.T) {
	groups := makeStandardGroups()
	byName := map[string]models.Group{}
	for _, g := range groups {
		byName[g.Name] = g
	}
	// A balanced-ish spread that leaves 국내배당 (domestic ETF, eligible) and 해외배당
	// (only the US-listed SCHD, ineligible for IRP) both UNDER their band.
	stocks := map[string]models.Stock{
		"국내성장": makeStockETF("069500", byName["국내성장"].ID),
		"해외성장": makeStockETF("133690", byName["해외성장"].ID),
		"해외안정": makeStockETF("360750", byName["해외안정"].ID),
		"국내배당": makeStockETF("000660", byName["국내배당"].ID), // domestic dividend ETF
		"해외배당": makeStockETF("SCHD", byName["해외배당"].ID),   // US-listed ETF only
	}
	// total 760; 국내배당 50 = 6.6% (<12) under, 해외배당 10 = 1.3% (<12) under;
	// 국내성장/해외성장/해외안정 over-band → sells fund the buys.
	summary := makeSummary(groups, stocks, map[string]numeric.Decimal{
		"국내성장": mustN("350"),
		"해외성장": mustN("250"),
		"해외안정": mustN("100"),
		"국내배당": mustN("50"),
		"해외배당": mustN("10"),
	})

	svc := services.NewRebalanceService()

	run := func(accountType string) (buyTickers []string, unmet []string) {
		acc := makeTypedAccount("ACC", "0", accountType)
		holdings := makeHoldingsByAccount([]models.Account{acc}, stocks, map[string]map[string]string{
			"ACC": {"국내성장": "350", "해외성장": "250", "해외안정": "100", "국내배당": "50", "해외배당": "10"},
		})
		plan, err := svc.BuildPlan(services.BuildPlanParams{
			Summary:           summary,
			Accounts:          []models.Account{acc},
			HoldingsByAccount: holdings,
			Groups:            groups,
			Stocks:            stockSlice(stocks),
		})
		if err != nil {
			t.Fatalf("BuildPlan(%s): %v", accountType, err)
		}
		for _, r := range plan.BuyRecs {
			buyTickers = append(buyTickers, r.Ticker)
		}
		for _, sum := range plan.AccountSummaries {
			unmet = append(unmet, sum.UnmetGroups...)
		}
		return buyTickers, unmet
	}

	// IRP: SCHD blocked, 해외배당 unmet, but eligible domestic 000660 still bought.
	irpBuys, irpUnmet := run(models.AccountTypeIRP)
	if containsString(irpBuys, "SCHD") {
		t.Errorf("IRP must not buy US-listed SCHD; buys = %v", irpBuys)
	}
	if !containsString(irpUnmet, "해외배당") {
		t.Errorf("IRP should report 해외배당 unmet; unmet = %v", irpUnmet)
	}
	if !containsString(irpBuys, "000660") {
		t.Errorf("IRP should still buy eligible domestic ETF 000660; buys = %v", irpBuys)
	}

	// Brokerage: SCHD is eligible and gets bought (control).
	bkBuys, _ := run(models.AccountTypeBrokerage)
	if !containsString(bkBuys, "SCHD") {
		t.Errorf("brokerage should buy SCHD; buys = %v", bkBuys)
	}
}

func mustN(s string) numeric.Decimal {
	d, err := numeric.FromString(s)
	if err != nil {
		panic("mustN: " + s + ": " + err.Error())
	}
	return d
}

func stockSlice(m map[string]models.Stock) []models.Stock {
	s := make([]models.Stock, 0, len(m))
	for _, v := range m {
		s = append(s, v)
	}
	return s
}

func numericEq(a numeric.Decimal, b string) bool {
	bv, _ := numeric.FromString(b)
	return a.Equal(bv.Decimal)
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func findAccountSummary(plan models.RebalancePlan, id uuidx.UUID) models.AccountRebalanceSummary {
	for _, s := range plan.AccountSummaries {
		if s.AccountID == id {
			return s
		}
	}
	panic("account summary not found: " + id.String())
}

// TestBuildPlanEmptyPortfolio verifies emptyPlan() is returned when total assets == 0.
func TestBuildPlanEmptyPortfolio(t *testing.T) {
	svc := services.NewRebalanceService()
	summary := models.PortfolioSummary{}
	plan, err := svc.BuildPlan(services.BuildPlanParams{
		Summary:  summary,
		Accounts: []models.Account{},
		Groups:   []models.Group{},
		Stocks:   []models.Stock{},
	})
	if err != nil {
		t.Fatalf("BuildPlan empty: %v", err)
	}
	if plan.GroupDiagnostics == nil {
		t.Error("emptyPlan() GroupDiagnostics should not be nil")
	}
	if len(plan.SellRecs) != 0 || len(plan.BuyRecs) != 0 {
		t.Error("empty plan should have no recommendations")
	}
}
