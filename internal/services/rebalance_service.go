package services

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
	"github.com/shopspring/decimal"
)

var (
	_percentBase = decimal.NewFromInt(100)
	_regionBand  = decimal.NewFromInt(5)
	_two         = decimal.NewFromInt(2)
)

var _groupOrder = []string{
	"국내성장",
	"국내배당",
	"해외성장",
	"해외안정",
	"해외배당",
}

var _groupBands = map[string]decimal.Decimal{
	"국내성장": decimal.NewFromInt(5),
	"국내배당": decimal.NewFromInt(3),
	"해외성장": decimal.NewFromInt(5),
	"해외안정": decimal.NewFromInt(2),
	"해외배당": decimal.NewFromInt(3),
}

// BuildPlanParams bundles inputs for BuildPlan.
type BuildPlanParams struct {
	Summary           models.PortfolioSummary
	Accounts          []models.Account
	HoldingsByAccount map[uuidx.UUID][]models.Holding
	Groups            []models.Group
	Stocks            []models.Stock
	RestrictOverseas  bool
}

// RebalanceService calculates v2.0 rebalance recommendations.
type RebalanceService struct{}

// NewRebalanceService creates a RebalanceService.
func NewRebalanceService() *RebalanceService { return &RebalanceService{} }

// BuildPlan computes sell/buy recommendations from pre-assembled inputs.
func (s *RebalanceService) BuildPlan(p BuildPlanParams) (models.RebalancePlan, error) {
	total := p.Summary.TotalAssets.Decimal
	if p.Summary.TotalAssets.IsZero() {
		total = p.Summary.TotalValue.Decimal
	}
	if !total.IsPositive() {
		return emptyPlan(), nil
	}

	groupByID := map[uuidx.UUID]models.Group{}
	for _, g := range p.Groups {
		groupByID[g.ID] = g
	}
	stockByID := map[uuidx.UUID]models.Stock{}
	for _, st := range p.Stocks {
		stockByID[st.ID] = st
	}
	accountByID := map[uuidx.UUID]models.Account{}
	for _, a := range p.Accounts {
		accountByID[a.ID] = a
	}

	targetByGroup := s.buildTargetByGroup(p.Groups)

	currentByGroup := map[string]decimal.Decimal{}
	for _, name := range _groupOrder {
		currentByGroup[name] = decimal.Zero
	}
	tickerSnapshots, err := s.buildTickerSnapshots(p.Summary)
	if err != nil {
		return models.RebalancePlan{}, err
	}
	for _, snap := range tickerSnapshots {
		currentByGroup[snap.rebalanceGroup] = currentByGroup[snap.rebalanceGroup].Add(snap.totalValueKRW)
	}

	groupDiags := s.buildGroupDiagnostics(currentByGroup, targetByGroup, total)
	regionDiag := s.buildRegionDiagnostic(currentByGroup, targetByGroup, total)

	positions, err := s.buildAccountPositions(p.HoldingsByAccount, accountByID, stockByID, groupByID, tickerSnapshots)
	if err != nil {
		return models.RebalancePlan{}, err
	}

	accountAUM := s.buildAccountAUM(p.Accounts, positions)
	targetByAccountGroup := s.planTargetsByAccountGroup(p.Accounts, accountAUM, targetByGroup)
	accountGroupState := s.buildAccountGroupState(p.Accounts, positions, accountAUM, targetByAccountGroup, targetByGroup)

	accountTypeByID := map[uuidx.UUID]*string{}
	availableTypes := map[string]bool{}
	for _, a := range p.Accounts {
		accountTypeByID[a.ID] = a.AccountType
		if a.AccountType != nil && models.ValidAccountType(*a.AccountType) {
			availableTypes[*a.AccountType] = true
		}
	}

	sellByAccountGroup := s.calcSellAmounts(p.Accounts, accountGroupState, accountAUM, positions, p.RestrictOverseas)
	sellRecs, sellCashByAccount, soldByAccountGroup, sellRecsByAccountID := s.buildSellRecs(
		sellByAccountGroup, positions, accountGroupState, accountTypeByID, availableTypes, p.RestrictOverseas,
	)

	buyRecs, unusedCashByAccount, unmetByAccount, buyRecsByAccountID := s.buildBuyRecs(
		p.Accounts, positions, accountGroupState, accountAUM,
		soldByAccountGroup, sellCashByAccount, tickerSnapshots, p.RestrictOverseas,
	)

	accountSummaries := s.buildAccountSummaries(
		p.Accounts, sellRecsByAccountID, buyRecsByAccountID,
		sellCashByAccount, unusedCashByAccount, unmetByAccount,
	)

	return models.RebalancePlan{
		SellRecs:         sellRecs,
		BuyRecs:          buyRecs,
		GroupDiagnostics: groupDiags,
		RegionDiagnostic: regionDiag,
		TotalAssetsKRW:   numeric.Wrap(total),
		AccountSummaries: accountSummaries,
	}, nil
}

// --- internal types ---

type tickerSnapshot struct {
	ticker          string
	rebalanceGroup  string
	sourceGroup     string
	currency        string
	stockName       string
	isETF           bool
	totalQty        decimal.Decimal
	totalValueLocal decimal.Decimal
	totalValueKRW   decimal.Decimal
}

type accountPosition struct {
	accountID      uuidx.UUID
	accountName    string
	ticker         string
	rebalanceGroup string
	sourceGroup    string
	currency       string
	stockName      string
	isETF          bool
	quantity       decimal.Decimal
	valueLocal     decimal.Decimal
	valueKRW       decimal.Decimal
}

type accountGroupState struct {
	currentValueKRW decimal.Decimal
	currentPct      decimal.Decimal
	targetPct       decimal.Decimal
	upperPct        decimal.Decimal
	lowerPct        decimal.Decimal
	targetValueKRW  decimal.Decimal
	// mirrorTargetValueKRW is the pre-tax-location uniform target (global group
	// target % × account AUM). Comparing targetValueKRW against it reveals whether
	// tax-location concentration pushed this group OUT of (≈0) or INTO this account.
	mirrorTargetValueKRW decimal.Decimal
	isUpperBreached      bool
	isLowerBreached      bool
}

type buyCandidate struct {
	ticker         string
	currency       string
	stockName      string
	sourceGroup    string
	rebalanceGroup string
	qtyBase        decimal.Decimal
	valueLocalBase decimal.Decimal
	valueKRWBase   decimal.Decimal
}

// --- helpers ---

func emptyPlan() models.RebalancePlan {
	return models.RebalancePlan{
		GroupDiagnostics: []models.GroupDiagnostic{},
		AccountSummaries: []models.AccountRebalanceSummary{},
	}
}

func toPercent(value, total decimal.Decimal) decimal.Decimal {
	if total.IsZero() {
		return decimal.Zero
	}
	return value.Div(total).Mul(_percentBase)
}

// toGroup normalises a group name (strips whitespace) and returns it if valid.
func toGroup(name string) (string, bool) {
	// strip all whitespace like Python's "".join(split())
	runes := []rune{}
	for _, r := range name {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			runes = append(runes, r)
		}
	}
	norm := string(runes)
	if _, ok := _groupBands[norm]; ok {
		return norm, true
	}
	return "", false
}

func (s *RebalanceService) buildTargetByGroup(groups []models.Group) map[string]decimal.Decimal {
	result := map[string]decimal.Decimal{}
	for _, name := range _groupOrder {
		result[name] = decimal.Zero
	}
	for _, g := range groups {
		rg, ok := toGroup(g.Name)
		if !ok {
			continue
		}
		result[rg] = result[rg].Add(decimal.NewFromFloat(g.TargetPercentage))
	}
	return result
}

func (s *RebalanceService) buildTickerSnapshots(summary models.PortfolioSummary) (map[string]*tickerSnapshot, error) {
	snaps := map[string]*tickerSnapshot{}
	for _, pair := range summary.Holdings {
		rg, ok := toGroup(pair.Group.Name)
		if !ok {
			return nil, fmt.Errorf("리밸런싱 그룹 매핑 불가: %s — 그룹명은 국내성장/국내배당/해외성장/해외안정/해외배당 중 하나여야 합니다", pair.Group.Name)
		}
		h := pair.Holding
		valueLocal := h.Price.Mul(h.Quantity.Decimal)
		var valueKRW decimal.Decimal
		if h.ValueKRW != nil {
			valueKRW = h.ValueKRW.Decimal
		} else {
			valueKRW = valueLocal
		}
		name := h.Name
		if name == "" {
			name = h.Stock.Ticker
		}
		ticker := h.Stock.Ticker
		if existing, ok := snaps[ticker]; ok {
			existing.totalQty = existing.totalQty.Add(h.Quantity.Decimal)
			existing.totalValueLocal = existing.totalValueLocal.Add(valueLocal)
			existing.totalValueKRW = existing.totalValueKRW.Add(valueKRW)
		} else {
			snaps[ticker] = &tickerSnapshot{
				ticker:          ticker,
				rebalanceGroup:  rg,
				sourceGroup:     pair.Group.Name,
				currency:        h.Currency,
				stockName:       name,
				isETF:           assetIsETF(h.Stock.AssetClass),
				totalQty:        h.Quantity.Decimal,
				totalValueLocal: valueLocal,
				totalValueKRW:   valueKRW,
			}
		}
	}
	return snaps, nil
}

func (s *RebalanceService) buildGroupDiagnostics(currentByGroup, targetByGroup map[string]decimal.Decimal, total decimal.Decimal) []models.GroupDiagnostic {
	diags := make([]models.GroupDiagnostic, 0, len(_groupOrder))
	for _, name := range _groupOrder {
		target := targetByGroup[name]
		band := _groupBands[name]
		currentVal := currentByGroup[name]
		current := toPercent(currentVal, total)
		lower := target.Sub(band)
		upper := target.Add(band)
		diags = append(diags, models.GroupDiagnostic{
			RebalanceGroupName: name,
			TargetPct:          numeric.Wrap(target),
			BandPct:            numeric.Wrap(band),
			LowerPct:           numeric.Wrap(lower),
			UpperPct:           numeric.Wrap(upper),
			CurrentPct:         numeric.Wrap(current),
			CurrentValueKRW:    numeric.Wrap(currentVal),
			IsUpperBreached:    current.GreaterThan(upper),
			IsLowerBreached:    current.LessThan(lower),
		})
	}
	return diags
}

func (s *RebalanceService) buildRegionDiagnostic(currentByGroup, targetByGroup map[string]decimal.Decimal, total decimal.Decimal) models.RegionDiagnostic {
	targetKR := targetByGroup["국내성장"].Add(targetByGroup["국내배당"])
	targetUS := targetByGroup["해외성장"].Add(targetByGroup["해외안정"]).Add(targetByGroup["해외배당"])
	currentKRVal := currentByGroup["국내성장"].Add(currentByGroup["국내배당"])
	currentKR := toPercent(currentKRVal, total)
	currentUS := _percentBase.Sub(currentKR)
	lowerKR := targetKR.Sub(_regionBand)
	upperKR := targetKR.Add(_regionBand)
	return models.RegionDiagnostic{
		TargetKRPct:  numeric.Wrap(targetKR),
		TargetUSPct:  numeric.Wrap(targetUS),
		CurrentKRPct: numeric.Wrap(currentKR),
		CurrentUSPct: numeric.Wrap(currentUS),
		LowerKRPct:   numeric.Wrap(lowerKR),
		UpperKRPct:   numeric.Wrap(upperKR),
		IsTriggered:  currentKR.LessThan(lowerKR) || currentKR.GreaterThan(upperKR),
	}
}

func (s *RebalanceService) buildAccountPositions(
	holdingsByAccount map[uuidx.UUID][]models.Holding,
	accountByID map[uuidx.UUID]models.Account,
	stockByID map[uuidx.UUID]models.Stock,
	groupByID map[uuidx.UUID]models.Group,
	snapshots map[string]*tickerSnapshot,
) ([]accountPosition, error) {
	var positions []accountPosition
	for accountID, holdings := range holdingsByAccount {
		account, ok := accountByID[accountID]
		if !ok {
			continue
		}
		for _, h := range holdings {
			if h.Quantity.IsZero() || !h.Quantity.IsPositive() {
				continue
			}
			stock, ok := stockByID[h.StockID]
			if !ok {
				continue
			}
			group, ok := groupByID[stock.GroupID]
			if !ok {
				continue
			}
			rg, ok := toGroup(group.Name)
			if !ok {
				return nil, fmt.Errorf("리밸런싱 그룹 매핑 불가: %s — 그룹명은 국내성장/국내배당/해외성장/해외안정/해외배당 중 하나여야 합니다", group.Name)
			}
			snap := snapshots[stock.Ticker]
			if snap == nil || snap.totalQty.IsZero() {
				continue
			}
			ratio := h.Quantity.Decimal.Div(snap.totalQty)
			positions = append(positions, accountPosition{
				accountID:      account.ID,
				accountName:    account.Name,
				ticker:         stock.Ticker,
				rebalanceGroup: rg,
				sourceGroup:    group.Name,
				currency:       snap.currency,
				stockName:      snap.stockName,
				isETF:          assetIsETF(stock.AssetClass),
				quantity:       h.Quantity.Decimal,
				valueLocal:     snap.totalValueLocal.Mul(ratio),
				valueKRW:       snap.totalValueKRW.Mul(ratio),
			})
		}
	}
	return positions, nil
}

func (s *RebalanceService) buildAccountAUM(accounts []models.Account, positions []accountPosition) map[uuidx.UUID]decimal.Decimal {
	posValByAccount := map[uuidx.UUID]decimal.Decimal{}
	for _, p := range positions {
		posValByAccount[p.accountID] = posValByAccount[p.accountID].Add(p.valueKRW)
	}
	result := map[uuidx.UUID]decimal.Decimal{}
	for _, a := range accounts {
		result[a.ID] = posValByAccount[a.ID].Add(a.CashBalance.Decimal)
	}
	return result
}

// _placementScore ranks how tax-preferred it is to place each rebalance group
// in each account type (higher = stronger preference). Korea, 2026-06. These
// are the contestable, adjustable tax-opinion knobs of the engine — see
// docs/adr/0001. Used only to DIRECT trades that a band breach already requires
// (which account sells / buys), never to force proactive relocation.
//
// 해외배당 (국내상장 해외배당 ETF): a general 위탁 account taxes both 매매차익 and
// 분배금 as 배당소득 15.4% and folds them into 금융소득종합과세. ISA shelters them
// (비과세 한도 후 9.9% 분리과세), and 연금/IRP defer to 연금소득세 3.3–5.5%. The 2025
// 선환급 폐지 removed the *foreign-withholding* refund (lost in every account
// type alike), so it does NOT make 위탁 preferable — ISA still wins, 연금/IRP
// next, 위탁 last. (Earlier this row leaned brokerage; corrected.)
// 국내성장 leans 위탁: domestic listed-stock gains are largely untaxed for retail,
// so it should not occupy scarce tax-advantaged space.
var _placementScore = map[string]map[string]int{
	"국내배당": {models.AccountTypeIRP: 10, models.AccountTypePension: 10, models.AccountTypeISA: 9, models.AccountTypeBrokerage: 2},
	"해외성장": {models.AccountTypeIRP: 8, models.AccountTypePension: 8, models.AccountTypeISA: 7, models.AccountTypeBrokerage: 5},
	"해외안정": {models.AccountTypeIRP: 7, models.AccountTypePension: 7, models.AccountTypeISA: 6, models.AccountTypeBrokerage: 5},
	"국내성장": {models.AccountTypeBrokerage: 6, models.AccountTypeISA: 4, models.AccountTypeIRP: 3, models.AccountTypePension: 3},
	"해외배당": {models.AccountTypeISA: 8, models.AccountTypeIRP: 6, models.AccountTypePension: 6, models.AccountTypeBrokerage: 4},
}

// planTargetsByAccountGroup computes the per-account, per-group target VALUE
// (KRW) that the rest of the engine drives toward. Instead of mirroring the
// global group target into every account, it concentrates each group in the
// account TYPE that holds it most tax-efficiently, subject to each account's
// FIXED AUM (no cross-account transfer — the engine only swaps within an
// account, so contribution caps / withdrawal locks cannot be violated).
//
// Allocation is done at the (group × accountType) level then split across the
// accounts of a type in proportion to AUM. Consequences:
//   - A single account type (e.g. all brokerage) reproduces the old uniform
//     mirror exactly (proportional split of the whole-portfolio target).
//   - Divergence only appears across DIFFERENT types — i.e. when tax-location
//     actually matters.
//   - nil / unrecognized account_type is routed to the uniform global target
//     (NOT zero) so an unclassified account is never told to liquidate.
//
// The planner does NOT enforce per-security eligibility (canHold) — that is the
// job of the buy guard in selectBuyCandidate*. An infeasible target simply
// surfaces as an unmet group, never an illegal buy.
func (s *RebalanceService) planTargetsByAccountGroup(
	accounts []models.Account,
	accountAUM map[uuidx.UUID]decimal.Decimal,
	targetByGroup map[string]decimal.Decimal,
) map[[2]string]decimal.Decimal {
	target := map[[2]string]decimal.Decimal{}

	totalAUM := decimal.Zero
	for _, a := range accounts {
		totalAUM = totalAUM.Add(accountAUM[a.ID])
	}
	if !totalAUM.IsPositive() {
		return target
	}

	groupRemaining := map[string]decimal.Decimal{}
	for _, g := range _groupOrder {
		groupRemaining[g] = targetByGroup[g].Div(_percentBase).Mul(totalAUM)
	}

	// Partition accounts: recognized type vs unclassified (nil / unknown).
	typeAccounts := map[string][]models.Account{}
	var untyped []models.Account
	for _, a := range accounts {
		if a.AccountType != nil && models.ValidAccountType(*a.AccountType) {
			t := *a.AccountType
			typeAccounts[t] = append(typeAccounts[t], a)
		} else {
			untyped = append(untyped, a)
		}
	}

	// Unclassified accounts keep the uniform global target; reserve their share.
	for _, a := range untyped {
		aum := accountAUM[a.ID]
		for _, g := range _groupOrder {
			v := targetByGroup[g].Div(_percentBase).Mul(aum)
			target[[2]string{a.ID.String(), g}] = v
			groupRemaining[g] = groupRemaining[g].Sub(v)
		}
	}

	// Capacity per recognized type.
	typeRemaining := map[string]decimal.Decimal{}
	for t, accs := range typeAccounts {
		cap := decimal.Zero
		for _, a := range accs {
			cap = cap.Add(accountAUM[a.ID])
		}
		typeRemaining[t] = cap
	}

	// Deterministic fill order: score desc, then _groupOrder index, then type.
	type cell struct {
		group string
		typ   string
		score int
		gidx  int
	}
	var cells []cell
	for gi, g := range _groupOrder {
		for t := range typeAccounts {
			cells = append(cells, cell{g, t, _placementScore[g][t], gi})
		}
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].score != cells[j].score {
			return cells[i].score > cells[j].score
		}
		if cells[i].gidx != cells[j].gidx {
			return cells[i].gidx < cells[j].gidx
		}
		return cells[i].typ < cells[j].typ
	})

	targetByType := map[[2]string]decimal.Decimal{} // (group, type) -> value
	for _, c := range cells {
		fill := decimal.Min(groupRemaining[c.group], typeRemaining[c.typ])
		if !fill.IsPositive() {
			continue
		}
		key := [2]string{c.group, c.typ}
		targetByType[key] = targetByType[key].Add(fill)
		groupRemaining[c.group] = groupRemaining[c.group].Sub(fill)
		typeRemaining[c.typ] = typeRemaining[c.typ].Sub(fill)
	}

	// Split each (group, type) value across that type's accounts ∝ AUM.
	for t, accs := range typeAccounts {
		typeCap := decimal.Zero
		for _, a := range accs {
			typeCap = typeCap.Add(accountAUM[a.ID])
		}
		if !typeCap.IsPositive() {
			continue
		}
		for _, g := range _groupOrder {
			tv := targetByType[[2]string{g, t}]
			if !tv.IsPositive() {
				continue
			}
			for _, a := range accs {
				share := accountAUM[a.ID].Div(typeCap)
				key := [2]string{a.ID.String(), g}
				target[key] = target[key].Add(tv.Mul(share))
			}
		}
	}
	return target
}

func (s *RebalanceService) buildAccountGroupState(
	accounts []models.Account,
	positions []accountPosition,
	accountAUM map[uuidx.UUID]decimal.Decimal,
	targetValueByAccountGroup map[[2]string]decimal.Decimal,
	targetByGroup map[string]decimal.Decimal,
) map[[2]string]accountGroupState {
	currentByAccountGroup := map[[2]string]decimal.Decimal{}
	for _, p := range positions {
		key := [2]string{p.accountID.String(), p.rebalanceGroup}
		currentByAccountGroup[key] = currentByAccountGroup[key].Add(p.valueKRW)
	}

	state := map[[2]string]accountGroupState{}
	for _, account := range accounts {
		aum := accountAUM[account.ID]
		for _, gname := range _groupOrder {
			key := [2]string{account.ID.String(), gname}
			currentVal := currentByAccountGroup[key]
			currentPct := toPercent(currentVal, aum)
			targetVal := targetValueByAccountGroup[key]
			targetPct := toPercent(targetVal, aum)
			mirrorTargetVal := targetByGroup[gname].Div(_percentBase).Mul(aum)
			band := _groupBands[gname]
			upperPct := targetPct.Add(band)
			lowerPct := targetPct.Sub(band)
			state[key] = accountGroupState{
				currentValueKRW:      currentVal,
				currentPct:           currentPct,
				targetPct:            targetPct,
				upperPct:             upperPct,
				lowerPct:             lowerPct,
				targetValueKRW:       targetVal,
				mirrorTargetValueKRW: mirrorTargetVal,
				isUpperBreached:      currentPct.GreaterThan(upperPct),
				isLowerBreached:      currentPct.LessThan(lowerPct),
			}
		}
	}
	return state
}

func (s *RebalanceService) calcSellAmounts(
	accounts []models.Account,
	state map[[2]string]accountGroupState,
	accountAUM map[uuidx.UUID]decimal.Decimal,
	positions []accountPosition,
	restrictOverseas bool,
) map[[2]string]decimal.Decimal {
	var eligibleKeys map[[2]string]bool
	if restrictOverseas {
		eligibleKeys = map[[2]string]bool{}
		for _, p := range positions {
			if isDomesticTicker(p.ticker) {
				eligibleKeys[[2]string{p.accountID.String(), p.rebalanceGroup}] = true
			}
		}
	}

	sell := map[[2]string]decimal.Decimal{}
	for _, account := range accounts {
		aum := accountAUM[account.ID]
		if !aum.IsPositive() {
			continue
		}
		for _, gname := range _groupOrder {
			key := [2]string{account.ID.String(), gname}
			if eligibleKeys != nil && !eligibleKeys[key] {
				continue
			}
			st := state[key]
			if !st.isUpperBreached {
				continue
			}
			midpoint := st.currentPct.Add(st.targetPct).Div(_two)
			nextWeight := decimal.Min(midpoint, st.upperPct)
			sellWeight := st.currentPct.Sub(nextWeight)
			if !sellWeight.IsPositive() {
				continue
			}
			sellKRW := sellWeight.Div(_percentBase).Mul(aum)
			sellKRW = decimal.Min(sellKRW, st.currentValueKRW)
			if sellKRW.IsPositive() {
				sell[key] = sellKRW
			}
		}
	}
	return sell
}

func (s *RebalanceService) buildSellRecs(
	sellByAccountGroup map[[2]string]decimal.Decimal,
	positions []accountPosition,
	state map[[2]string]accountGroupState,
	accountTypeByID map[uuidx.UUID]*string,
	availableTypes map[string]bool,
	restrictOverseas bool,
) (
	[]models.RebalanceRecommendation,
	map[uuidx.UUID]decimal.Decimal,
	map[[2]string]decimal.Decimal,
	map[uuidx.UUID][]models.RebalanceRecommendation,
) {
	var recs []models.RebalanceRecommendation
	sellCashByAccount := map[uuidx.UUID]decimal.Decimal{}
	soldByAccountGroup := map[[2]string]decimal.Decimal{}
	recsByAccountID := map[uuidx.UUID][]models.RebalanceRecommendation{}

	for _, gname := range _groupOrder {
		// collect account→sellAmount for this group, sorted by account UUID string
		type entry struct {
			accountID uuidx.UUID
			sellKRW   decimal.Decimal
		}
		var entries []entry
		for key, sellKRW := range sellByAccountGroup {
			if key[1] == gname && sellKRW.IsPositive() {
				id, _ := uuidx.Parse(key[0])
				entries = append(entries, entry{id, sellKRW})
			}
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].accountID.String() < entries[j].accountID.String()
		})

		for _, e := range entries {
			targetSell := e.sellKRW
			key := [2]string{e.accountID.String(), gname}
			st := state[key]

			accountPositions := filterPositions(positions, func(p accountPosition) bool {
				return p.accountID == e.accountID &&
					p.rebalanceGroup == gname &&
					p.valueKRW.IsPositive() &&
					(!restrictOverseas || isDomesticTicker(p.ticker))
			})
			if len(accountPositions) == 0 {
				continue
			}

			accountTotal := sumValueKRW(accountPositions)
			targetSell = decimal.Min(targetSell, accountTotal)
			if !targetSell.IsPositive() {
				continue
			}

			// sell sort: domestic first (1 if domestic → domestic prioritised in sell)
			sort.Slice(accountPositions, func(i, j int) bool {
				di := domKey(accountPositions[i].ticker)
				dj := domKey(accountPositions[j].ticker)
				if di != dj {
					return di < dj // domestic (0 mapped below) vs overseas (1 mapped below)
				}
				if !accountPositions[i].valueKRW.Equal(accountPositions[j].valueKRW) {
					return accountPositions[i].valueKRW.GreaterThan(accountPositions[j].valueKRW)
				}
				return accountPositions[i].ticker < accountPositions[j].ticker
			})
			// Python sell sort: key=(1 if domestic else 0, -value, ticker) → overseas first
			// "1 if domestic" means overseas=0 sorts before domestic=1
			// We need to REVERSE the domestic flag for sell: overseas sorts first
			sort.SliceStable(accountPositions, func(i, j int) bool {
				// sell: overseas first (0), domestic second (1)
				si := sellDomKey(accountPositions[i].ticker)
				sj := sellDomKey(accountPositions[j].ticker)
				if si != sj {
					return si < sj
				}
				if !accountPositions[i].valueKRW.Equal(accountPositions[j].valueKRW) {
					return accountPositions[i].valueKRW.GreaterThan(accountPositions[j].valueKRW)
				}
				return accountPositions[i].ticker < accountPositions[j].ticker
			})

			remaining := targetSell
			for i, pos := range accountPositions {
				if !remaining.IsPositive() {
					break
				}
				isLast := i == len(accountPositions)-1
				var sellKRW decimal.Decimal
				if isLast {
					sellKRW = decimal.Min(remaining, pos.valueKRW)
				} else {
					proportional := targetSell.Mul(pos.valueKRW.Div(accountTotal))
					sellKRW = decimal.Min(proportional, decimal.Min(remaining, pos.valueKRW))
				}
				if !sellKRW.IsPositive() {
					continue
				}

				amountLocal := krwToLocal(sellKRW, pos.currency, pos.valueLocal, pos.valueKRW)
				qty := calcQuantity(amountLocal, pos.valueLocal, pos.quantity)

				rec := models.RebalanceRecommendation{
					Ticker:             pos.ticker,
					Action:             models.ActionSell,
					Amount:             numeric.Wrap(amountLocal),
					Priority:           len(recs) + 1,
					Currency:           pos.currency,
					Quantity:           ptrDecimal(qty),
					StockName:          pos.stockName,
					GroupName:          pos.sourceGroup,
					AccountName:        pos.accountName,
					RebalanceGroupName: gname,
					Reason:             sellReason(st, gname, accountTypeByID[e.accountID], availableTypes),
					TriggerType:        "group",
					AmountKRW:          numeric.Wrap(sellKRW),
					AmountLocal:        numeric.Wrap(amountLocal),
				}
				recs = append(recs, rec)
				recsByAccountID[e.accountID] = append(recsByAccountID[e.accountID], rec)
				remaining = remaining.Sub(sellKRW)
				sellCashByAccount[e.accountID] = sellCashByAccount[e.accountID].Add(sellKRW)
				soldByAccountGroup[key] = soldByAccountGroup[key].Add(sellKRW)
			}
		}
	}
	return recs, sellCashByAccount, soldByAccountGroup, recsByAccountID
}

func (s *RebalanceService) buildBuyRecs(
	accounts []models.Account,
	positions []accountPosition,
	state map[[2]string]accountGroupState,
	accountAUM map[uuidx.UUID]decimal.Decimal,
	soldByAccountGroup map[[2]string]decimal.Decimal,
	sellCashByAccount map[uuidx.UUID]decimal.Decimal,
	snapshots map[string]*tickerSnapshot,
	restrictOverseas bool,
) (
	[]models.RebalanceRecommendation,
	map[uuidx.UUID]decimal.Decimal,
	map[uuidx.UUID][]string,
	map[uuidx.UUID][]models.RebalanceRecommendation,
) {
	var recs []models.RebalanceRecommendation
	unusedCashByAccount := map[uuidx.UUID]decimal.Decimal{}
	unmetByAccount := map[uuidx.UUID][]string{}
	recsByAccountID := map[uuidx.UUID][]models.RebalanceRecommendation{}

	for _, account := range accounts {
		cash := account.CashBalance.Decimal.Add(sellCashByAccount[account.ID])
		aum := accountAUM[account.ID]

		accountNeed := map[string]decimal.Decimal{}
		accountProjected := map[string]decimal.Decimal{}
		for _, gname := range _groupOrder {
			key := [2]string{account.ID.String(), gname}
			st := state[key]
			sold := soldByAccountGroup[key]
			projected := st.currentValueKRW.Sub(sold)
			accountNeed[gname] = decimal.Max(decimal.Zero, st.targetValueKRW.Sub(projected))
			accountProjected[gname] = projected
		}

		var unmetGroups []string
		blockedGroups := map[string]bool{}

		for cash.IsPositive() {
			gname := s.pickNextGroup(accountNeed, blockedGroups)
			if gname == "" {
				break
			}
			need := accountNeed[gname]
			if !need.IsPositive() {
				blockedGroups[gname] = true
				continue
			}

			candidate := s.selectBuyCandidateAccountScoped(account.ID, gname, positions, restrictOverseas, account.AccountType)
			if candidate == nil {
				candidate = s.selectBuyCandidatePortfolioFallback(gname, snapshots, restrictOverseas, account.AccountType)
			}
			if candidate == nil {
				unmetGroups = append(unmetGroups, gname)
				blockedGroups[gname] = true
				continue
			}

			buyKRW := decimal.Min(cash, need)
			if !buyKRW.IsPositive() {
				break
			}

			amountLocal := krwToLocal(buyKRW, candidate.currency, candidate.valueLocalBase, candidate.valueKRWBase)
			qty := calcQuantity(amountLocal, candidate.valueLocalBase, candidate.qtyBase)

			key := [2]string{account.ID.String(), gname}
			st := state[key]
			projectedPct := toPercent(accountProjected[gname], aum)

			rec := models.RebalanceRecommendation{
				Ticker:             candidate.ticker,
				Action:             models.ActionBuy,
				Amount:             numeric.Wrap(amountLocal),
				Priority:           len(recs) + 1,
				Currency:           candidate.currency,
				Quantity:           ptrDecimal(qty),
				StockName:          candidate.stockName,
				GroupName:          candidate.sourceGroup,
				AccountName:        account.Name,
				RebalanceGroupName: gname,
				Reason:             buyReason(st, projectedPct, gname, account.AccountType),
				TriggerType:        "group",
				AmountKRW:          numeric.Wrap(buyKRW),
				AmountLocal:        numeric.Wrap(amountLocal),
			}
			recs = append(recs, rec)
			recsByAccountID[account.ID] = append(recsByAccountID[account.ID], rec)

			cash = cash.Sub(buyKRW)
			accountProjected[gname] = accountProjected[gname].Add(buyKRW)
			accountNeed[gname] = decimal.Max(decimal.Zero, accountNeed[gname].Sub(buyKRW))
		}

		// collect remaining unmet groups (positive need, no candidate found)
		for _, gname := range _groupOrder {
			if blockedGroups[gname] {
				continue
			}
			if accountNeed[gname].IsPositive() {
				candidate := s.selectBuyCandidateAccountScoped(account.ID, gname, positions, restrictOverseas, account.AccountType)
				if candidate == nil {
					candidate = s.selectBuyCandidatePortfolioFallback(gname, snapshots, restrictOverseas, account.AccountType)
				}
				if candidate == nil && !containsStr(unmetGroups, gname) {
					unmetGroups = append(unmetGroups, gname)
				}
			}
		}

		unusedCashByAccount[account.ID] = cash
		unmetByAccount[account.ID] = unmetGroups
	}
	return recs, unusedCashByAccount, unmetByAccount, recsByAccountID
}

func (s *RebalanceService) pickNextGroup(need map[string]decimal.Decimal, blocked map[string]bool) string {
	type entry struct {
		name  string
		need  decimal.Decimal
		index int
	}
	var candidates []entry
	for i, name := range _groupOrder {
		if blocked[name] {
			continue
		}
		n := need[name]
		if n.IsPositive() {
			candidates = append(candidates, entry{name, n, i})
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.Slice(candidates, func(i, j int) bool {
		ci, cj := candidates[i], candidates[j]
		cmp := ci.need.Cmp(cj.need)
		if cmp != 0 {
			return cmp > 0 // highest need first
		}
		return ci.index > cj.index // tie: later in _groupOrder first (reverse index)
	})
	return candidates[0].name
}

func (s *RebalanceService) selectBuyCandidateAccountScoped(
	accountID uuidx.UUID,
	group string,
	positions []accountPosition,
	restrictOverseas bool,
	accountType *string,
) *buyCandidate {
	acc := filterPositions(positions, func(p accountPosition) bool {
		return p.accountID == accountID &&
			p.rebalanceGroup == group &&
			p.valueLocal.IsPositive() &&
			(!restrictOverseas || isDomesticTicker(p.ticker)) &&
			canHold(accountType, p.ticker, p.isETF)
	})
	if len(acc) == 0 {
		return nil
	}
	sort.Slice(acc, func(i, j int) bool {
		di, dj := domKey(acc[i].ticker), domKey(acc[j].ticker)
		if di != dj {
			return di < dj
		}
		if !acc[i].valueKRW.Equal(acc[j].valueKRW) {
			return acc[i].valueKRW.GreaterThan(acc[j].valueKRW)
		}
		return acc[i].ticker < acc[j].ticker
	})
	top := acc[0]
	return &buyCandidate{
		ticker:         top.ticker,
		currency:       top.currency,
		stockName:      top.stockName,
		sourceGroup:    top.sourceGroup,
		rebalanceGroup: top.rebalanceGroup,
		qtyBase:        top.quantity,
		valueLocalBase: top.valueLocal,
		valueKRWBase:   top.valueKRW,
	}
}

func (s *RebalanceService) selectBuyCandidatePortfolioFallback(
	group string,
	snapshots map[string]*tickerSnapshot,
	restrictOverseas bool,
	accountType *string,
) *buyCandidate {
	var snaps []*tickerSnapshot
	for _, snap := range snapshots {
		if snap.rebalanceGroup == group &&
			snap.totalValueLocal.IsPositive() &&
			(!restrictOverseas || isDomesticTicker(snap.ticker)) &&
			canHold(accountType, snap.ticker, snap.isETF) {
			snaps = append(snaps, snap)
		}
	}
	if len(snaps) == 0 {
		return nil
	}
	sort.Slice(snaps, func(i, j int) bool {
		di, dj := domKey(snaps[i].ticker), domKey(snaps[j].ticker)
		if di != dj {
			return di < dj
		}
		if !snaps[i].totalValueKRW.Equal(snaps[j].totalValueKRW) {
			return snaps[i].totalValueKRW.GreaterThan(snaps[j].totalValueKRW)
		}
		return snaps[i].ticker < snaps[j].ticker
	})
	top := snaps[0]
	return &buyCandidate{
		ticker:         top.ticker,
		currency:       top.currency,
		stockName:      top.stockName,
		sourceGroup:    top.sourceGroup,
		rebalanceGroup: top.rebalanceGroup,
		qtyBase:        top.totalQty,
		valueLocalBase: top.totalValueLocal,
		valueKRWBase:   top.totalValueKRW,
	}
}

func (s *RebalanceService) buildAccountSummaries(
	accounts []models.Account,
	sellRecsByAccountID map[uuidx.UUID][]models.RebalanceRecommendation,
	buyRecsByAccountID map[uuidx.UUID][]models.RebalanceRecommendation,
	sellCashByAccount map[uuidx.UUID]decimal.Decimal,
	unusedCashByAccount map[uuidx.UUID]decimal.Decimal,
	unmetByAccount map[uuidx.UUID][]string,
) []models.AccountRebalanceSummary {
	summaries := make([]models.AccountRebalanceSummary, 0, len(accounts))
	for _, account := range accounts {
		sellRecs := sellRecsByAccountID[account.ID]
		buyRecs := buyRecsByAccountID[account.ID]
		sellCash := sellCashByAccount[account.ID]
		totalBuy := decimal.Zero
		for _, r := range buyRecs {
			totalBuy = totalBuy.Add(r.AmountKRW.Decimal)
		}
		unused := unusedCashByAccount[account.ID]
		if _, ok := unusedCashByAccount[account.ID]; !ok {
			unused = account.CashBalance.Decimal
		}
		unmet := unmetByAccount[account.ID]
		summaries = append(summaries, models.AccountRebalanceSummary{
			AccountID:       account.ID,
			AccountName:     account.Name,
			StartingCashKRW: account.CashBalance,
			SellCashKRW:     numeric.Wrap(sellCash),
			TotalBuyKRW:     numeric.Wrap(totalBuy),
			UnusedCashKRW:   numeric.Wrap(unused),
			UnmetGroups:     unmet,
			SellRecs:        sellRecs,
			BuyRecs:         buyRecs,
		})
	}
	return summaries
}

// --- pure math helpers ---

func krwToLocal(amountKRW decimal.Decimal, currency string, valueLocal, valueKRW decimal.Decimal) decimal.Decimal {
	if currency == "USD" && valueLocal.IsPositive() && valueKRW.IsPositive() {
		fx := valueKRW.Div(valueLocal)
		if fx.IsPositive() {
			return amountKRW.Div(fx)
		}
	}
	return amountKRW
}

func calcQuantity(amountLocal, posLocalValue, posQty decimal.Decimal) *decimal.Decimal {
	if !posLocalValue.IsPositive() || !posQty.IsPositive() {
		return nil
	}
	q := amountLocal.Div(posLocalValue).Mul(posQty)
	return &q
}

func ptrDecimal(d *decimal.Decimal) *numeric.Decimal {
	if d == nil {
		return nil
	}
	v := numeric.Wrap(*d)
	return &v
}

// domKey returns 0 for domestic, 1 for overseas (buy sort: domestic first).
func domKey(ticker string) int {
	if isDomesticTicker(ticker) {
		return 0
	}
	return 1
}

// sellDomKey returns 0 for overseas, 1 for domestic (sell sort: overseas first).
func sellDomKey(ticker string) int {
	if isDomesticTicker(ticker) {
		return 1
	}
	return 0
}

func filterPositions(positions []accountPosition, pred func(accountPosition) bool) []accountPosition {
	var out []accountPosition
	for _, p := range positions {
		if pred(p) {
			out = append(out, p)
		}
	}
	return out
}

func sumValueKRW(positions []accountPosition) decimal.Decimal {
	sum := decimal.Zero
	for _, p := range positions {
		sum = sum.Add(p.valueKRW)
	}
	return sum
}

func floatOf(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}

// --- recommendation reasons (explain the WHY, incl. tax-location) ---

var (
	_taxPushOutRatio = decimal.NewFromFloat(0.5) // target ≤ 50% of uniform share ⇒ pushed out
	_taxPullInRatio  = decimal.NewFromFloat(1.5) // target ≥ 150% of uniform share ⇒ pulled in
)

// isTaxPushedOut reports whether tax-location concentration drove this account's
// target for the group well below its uniform (pre-tax-location) share — i.e. the
// group is held more tax-efficiently elsewhere, so this account is being emptied.
func isTaxPushedOut(st accountGroupState) bool {
	mirror := st.mirrorTargetValueKRW
	return mirror.IsPositive() && st.targetValueKRW.LessThan(mirror.Mul(_taxPushOutRatio))
}

// isTaxPulledIn reports whether tax-location concentration drove this account's
// target for the group well above its uniform share — i.e. this account type
// holds the group most tax-efficiently, so the group is concentrated here.
func isTaxPulledIn(st accountGroupState) bool {
	mirror := st.mirrorTargetValueKRW
	if !mirror.IsPositive() {
		// targetValueKRW and mirrorTargetValueKRW both derive from
		// (group target% × account AUM) in planTargetsByAccountGroup, so a
		// zero mirror implies a zero target — there is nothing to pull in.
		// Symmetric with isTaxPushedOut's mirror.IsPositive() guard.
		return false
	}
	return st.targetValueKRW.GreaterThan(mirror.Mul(_taxPullInRatio))
}

func sellReason(st accountGroupState, gname string, accountType *string, availableTypes map[string]bool) string {
	if isTaxPushedOut(st) {
		// Capacity-yield case: this account's own type is itself a tax-home for the
		// group (preferred), yet its target was still driven down — its scarce
		// capacity was yielded to an even higher-scored group held here. Group
		// targets hold in aggregate, so this group's target is met in other
		// accounts. The plain push-out wording would nonsensically name this very
		// account as the preferred destination, so reframe.
		if accountType != nil && isPreferredType(gname, *accountType, availableTypes) {
			if winner := higherPriorityGroup(gname, *accountType); winner != "" {
				return fmt.Sprintf(
					"세금 위치 최적화 — 이 계좌(%s)는 %s의 세금 효율이 더 높아 해당 그룹을 우선 배정하고, %s은(는) 비중을 줄여 다른 계좌에서 목표를 충족합니다 (현재 %.2f%% → 이 계좌 목표 %.2f%%)",
					accountTypeLabel(accountType), winner, gname,
					floatOf(st.currentPct), floatOf(st.targetPct))
			}
			return fmt.Sprintf(
				"세금 위치 최적화 — 이 계좌(%s) 용량을 세금 효율이 더 높은 그룹에 우선 배정하여 %s 비중을 축소합니다 (다른 계좌에서 목표를 충족, 현재 %.2f%% → 이 계좌 목표 %.2f%%)",
				accountTypeLabel(accountType), gname,
				floatOf(st.currentPct), floatOf(st.targetPct))
		}
		return fmt.Sprintf(
			"세금 위치 최적화 — %s은(는) %s에서 세금 효율이 높아 이 계좌(%s) 비중을 축소합니다 (현재 %.2f%% → 목표 %.2f%%)",
			gname, preferredAccountTypesLabel(gname, availableTypes), accountTypeLabel(accountType),
			floatOf(st.currentPct), floatOf(st.targetPct))
	}
	return fmt.Sprintf("과열 그룹 절반 감축 (%.2f%% -> 목표근접, 상단 %.2f%%)",
		floatOf(st.currentPct), floatOf(st.upperPct))
}

func buyReason(st accountGroupState, projectedPct decimal.Decimal, gname string, accountType *string) string {
	if isTaxPulledIn(st) {
		return fmt.Sprintf(
			"세금 위치 최적화 — %s 계좌가 %s의 세금 효율이 가장 높아 집중 매수합니다 (목표 %.2f%%)",
			accountTypeLabel(accountType), gname, floatOf(st.targetPct))
	}
	return fmt.Sprintf("목표 대비 부족분 보충 (%.2f%% -> 목표 %.2f%%)",
		floatOf(projectedPct), floatOf(st.targetPct))
}

// accountTypeLabel maps an account type code to a Korean display label.
func accountTypeLabel(t *string) string {
	if t == nil {
		return "미분류"
	}
	return accountTypeLabelStr(*t)
}

func accountTypeLabelStr(t string) string {
	switch t {
	case models.AccountTypeBrokerage:
		return "위탁"
	case models.AccountTypeIRP:
		return "IRP"
	case models.AccountTypePension:
		return "연금저축"
	case models.AccountTypeISA:
		return "ISA"
	default:
		return "미분류"
	}
}

// preferredAccountTypesLabel returns a "·"-joined label of the account type(s)
// that hold `group` most tax-efficiently (highest _placementScore). The max is
// taken over the account types the user actually has (availableTypes); an empty
// set falls back to all types. This keeps the label honest — e.g. 국내배당
// globally prefers IRP·연금, but for a user holding only {위탁, ISA} it must name
// ISA, since that is where the engine actually concentrated the group.
func preferredAccountTypesLabel(group string, availableTypes map[string]bool) string {
	scores := _placementScore[group]
	order := []string{
		models.AccountTypeIRP, models.AccountTypePension,
		models.AccountTypeISA, models.AccountTypeBrokerage,
	}
	consider := func(t string) bool {
		return len(availableTypes) == 0 || availableTypes[t]
	}
	maxScore := 0
	for _, t := range order {
		if consider(t) && scores[t] > maxScore {
			maxScore = scores[t]
		}
	}
	var labels []string
	for _, t := range order {
		if consider(t) && scores[t] == maxScore {
			labels = append(labels, accountTypeLabelStr(t))
		}
	}
	return strings.Join(labels, "·")
}

// isPreferredType reports whether accountType is among the highest-_placementScore
// types for group, restricted to the types the user actually holds. True means
// this account type is itself a tax-home for the group.
func isPreferredType(group, accountType string, availableTypes map[string]bool) bool {
	scores := _placementScore[group]
	maxScore := 0
	for t, sc := range scores {
		if (len(availableTypes) == 0 || availableTypes[t]) && sc > maxScore {
			maxScore = sc
		}
	}
	return scores[accountType] == maxScore
}

// higherPriorityGroup returns the group most strongly preferred in accountType
// that outranks `group` there — i.e. the group that won this account's scarce
// capacity. Returns "" when nothing outranks the group in this account type.
func higherPriorityGroup(group, accountType string) string {
	best := ""
	bestScore := _placementScore[group][accountType]
	for _, g := range _groupOrder {
		if g == group {
			continue
		}
		if sc := _placementScore[g][accountType]; sc > bestScore {
			best, bestScore = g, sc
		}
	}
	return best
}

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// isDomesticTicker mirrors kis.IsDomesticTicker without creating an import cycle.
// A 6-digit numeric string is a KOSPI/KOSDAQ ticker (domestic-listed).
func isDomesticTicker(ticker string) bool { return len(ticker) == 6 }

// assetIsETF reports whether a stock's asset_class marks it as an ETF.
// nil/unclassified → false (treated strictly: not an ETF).
func assetIsETF(assetClass *string) bool {
	return assetClass != nil && *assetClass == "etf"
}

// canHold reports whether an account of the given type may *buy* a security with
// the given listing (ticker) and asset class. Korean tax-account eligibility:
//   - brokerage (위탁): anything
//   - IRP / 연금저축: domestic-listed ETFs/funds only (no individual stocks,
//     no foreign-listed securities)
//   - ISA (중개형): domestic-listed only (ETF or individual stock)
//   - nil/unknown: blocked (strict — classify the account first)
func canHold(accountType *string, ticker string, isETF bool) bool {
	if accountType == nil {
		return false
	}
	switch *accountType {
	case models.AccountTypeBrokerage:
		return true
	case models.AccountTypeIRP, models.AccountTypePension:
		return isETF && isDomesticTicker(ticker)
	case models.AccountTypeISA:
		return isDomesticTicker(ticker)
	default:
		return false
	}
}

// ensure big is imported for potential future use (shopspring/decimal uses it internally)
var _ = big.NewInt
