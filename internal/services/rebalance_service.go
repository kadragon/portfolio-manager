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

	accountTypeByID := map[uuidx.UUID]*string{}
	availableTypes := map[string]bool{}
	for _, a := range p.Accounts {
		accountTypeByID[a.ID] = a.AccountType
		if a.AccountType != nil && models.ValidAccountType(*a.AccountType) {
			availableTypes[*a.AccountType] = true
		}
	}

	// Trade only what the AGGREGATE (all-accounts) band requires: sell groups above
	// their upper band down to target, buy groups below their lower band up to
	// target. In-band groups are never touched, so a portfolio already within band
	// produces no trades. _placementScore only DIRECTS which account sells/buys —
	// tax-location is reached gradually, riding the trades a breach already forces,
	// never a proactive relocation that would realize capital-gains tax.
	aggByGroup := buildGroupAggregates(currentByGroup, targetByGroup, total)
	sellNeedByGroup, buyNeedByGroup := computeGroupNetActions(aggByGroup)

	sellByAccountGroup := s.allocateSells(p.Accounts, positions, sellNeedByGroup, accountTypeByID, p.RestrictOverseas)
	sellRecs, sellCashByAccount, sellRecsByAccountID := s.buildSellRecs(
		sellByAccountGroup, positions, aggByGroup, accountTypeByID, availableTypes, p.RestrictOverseas,
	)

	buyRecs, unusedCashByAccount, unmetByAccount, buyRecsByAccountID := s.buildBuyRecs(
		p.Accounts, positions, buyNeedByGroup, sellCashByAccount,
		tickerSnapshots, aggByGroup, accountTypeByID, availableTypes, p.RestrictOverseas,
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

// groupAgg holds a rebalance group's portfolio-wide (aggregate) position vs its
// target band — the only level at which the engine decides whether to trade.
type groupAgg struct {
	currentValueKRW decimal.Decimal
	targetValueKRW  decimal.Decimal
	currentPct      decimal.Decimal
	targetPct       decimal.Decimal
	bandPct         decimal.Decimal
	isUpperBreached bool
	isLowerBreached bool
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

// buildGroupAggregates computes each group's portfolio-wide position vs its
// target band. This is the ONLY level at which the engine decides to trade.
func buildGroupAggregates(currentByGroup, targetByGroup map[string]decimal.Decimal, total decimal.Decimal) map[string]groupAgg {
	agg := map[string]groupAgg{}
	for _, g := range _groupOrder {
		currentVal := currentByGroup[g]
		targetPct := targetByGroup[g]
		band := _groupBands[g]
		currentPct := toPercent(currentVal, total)
		targetVal := targetPct.Div(_percentBase).Mul(total)
		agg[g] = groupAgg{
			currentValueKRW: currentVal,
			targetValueKRW:  targetVal,
			currentPct:      currentPct,
			targetPct:       targetPct,
			bandPct:         band,
			isUpperBreached: currentPct.GreaterThan(targetPct.Add(band)),
			isLowerBreached: currentPct.LessThan(targetPct.Sub(band)),
		}
	}
	return agg
}

// computeGroupNetActions turns aggregate band breaches into portfolio-level net
// trade amounts (KRW): an over-band group is sold down to TARGET, an under-band
// group is bought up to TARGET. In-band groups produce nothing.
func computeGroupNetActions(agg map[string]groupAgg) (sellNeed, buyNeed map[string]decimal.Decimal) {
	sellNeed = map[string]decimal.Decimal{}
	buyNeed = map[string]decimal.Decimal{}
	for _, g := range _groupOrder {
		a := agg[g]
		switch {
		case a.isUpperBreached:
			if d := a.currentValueKRW.Sub(a.targetValueKRW); d.IsPositive() {
				sellNeed[g] = d
			}
		case a.isLowerBreached:
			if d := a.targetValueKRW.Sub(a.currentValueKRW); d.IsPositive() {
				buyNeed[g] = d
			}
		}
	}
	return sellNeed, buyNeed
}

// taxAdvantaged reports whether selling in this account type realizes NO capital
// gains tax (연금/IRP/ISA defer or shelter; nil/위탁 are taxable). Used to bias
// sells away from taxable accounts so a band-breach rebalance does not trigger
// avoidable 양도세/배당소득세.
func taxAdvantaged(accountType *string) bool {
	if accountType == nil {
		return false
	}
	switch *accountType {
	case models.AccountTypeIRP, models.AccountTypePension, models.AccountTypeISA:
		return true
	default:
		return false
	}
}

// allocateSells distributes each over-band group's portfolio-level sell amount
// across the accounts that hold it. Account order, per group:
//  1. tax-advantaged accounts first (selling there realizes no gains tax),
//  2. then where the group is least tax-appropriate (ascending _placementScore —
//     i.e. move it out of accounts it shouldn't occupy, nudging placement),
//  3. larger holding first, then accountID for determinism.
//
// Selling is never blocked by eligibility (canHold guards buys only); RestrictOverseas
// still skips overseas tickers.
func (s *RebalanceService) allocateSells(
	accounts []models.Account,
	positions []accountPosition,
	sellNeedByGroup map[string]decimal.Decimal,
	accountTypeByID map[uuidx.UUID]*string,
	restrictOverseas bool,
) map[[2]string]decimal.Decimal {
	heldByAccountGroup := map[[2]string]decimal.Decimal{}
	for _, p := range positions {
		if restrictOverseas && !isDomesticTicker(p.ticker) {
			continue
		}
		key := [2]string{p.accountID.String(), p.rebalanceGroup}
		heldByAccountGroup[key] = heldByAccountGroup[key].Add(p.valueKRW)
	}

	sell := map[[2]string]decimal.Decimal{}
	for _, gname := range _groupOrder {
		need := sellNeedByGroup[gname]
		if !need.IsPositive() {
			continue
		}
		type cand struct {
			id           uuidx.UUID
			held         decimal.Decimal
			taxAdvantage bool
			score        int
		}
		var cands []cand
		for _, a := range accounts {
			held := heldByAccountGroup[[2]string{a.ID.String(), gname}]
			if !held.IsPositive() {
				continue
			}
			at := accountTypeByID[a.ID]
			sc := 0
			if at != nil {
				sc = _placementScore[gname][*at]
			}
			cands = append(cands, cand{a.ID, held, taxAdvantaged(at), sc})
		}
		sort.Slice(cands, func(i, j int) bool {
			if cands[i].taxAdvantage != cands[j].taxAdvantage {
				return cands[i].taxAdvantage // tax-advantaged (no realized tax) first
			}
			if cands[i].score != cands[j].score {
				return cands[i].score < cands[j].score // least tax-appropriate first
			}
			if !cands[i].held.Equal(cands[j].held) {
				return cands[i].held.GreaterThan(cands[j].held)
			}
			return cands[i].id.String() < cands[j].id.String()
		})

		remaining := need
		for _, c := range cands {
			if !remaining.IsPositive() {
				break
			}
			take := decimal.Min(remaining, c.held)
			if !take.IsPositive() {
				continue
			}
			sell[[2]string{c.id.String(), gname}] = take
			remaining = remaining.Sub(take)
		}
	}
	return sell
}

func (s *RebalanceService) buildSellRecs(
	sellByAccountGroup map[[2]string]decimal.Decimal,
	positions []accountPosition,
	aggByGroup map[string]groupAgg,
	accountTypeByID map[uuidx.UUID]*string,
	availableTypes map[string]bool,
	restrictOverseas bool,
) (
	[]models.RebalanceRecommendation,
	map[uuidx.UUID]decimal.Decimal,
	map[uuidx.UUID][]models.RebalanceRecommendation,
) {
	var recs []models.RebalanceRecommendation
	sellCashByAccount := map[uuidx.UUID]decimal.Decimal{}
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
					Reason:             sellReason(aggByGroup[gname], gname, accountTypeByID[e.accountID], availableTypes),
					TriggerType:        "group",
					AmountKRW:          numeric.Wrap(sellKRW),
					AmountLocal:        numeric.Wrap(amountLocal),
				}
				recs = append(recs, rec)
				recsByAccountID[e.accountID] = append(recsByAccountID[e.accountID], rec)
				remaining = remaining.Sub(sellKRW)
				sellCashByAccount[e.accountID] = sellCashByAccount[e.accountID].Add(sellKRW)
			}
		}
	}
	return recs, sellCashByAccount, recsByAccountID
}

// buildBuyRecs deploys each account's cash (starting cash + sell proceeds) toward
// the groups that are UNDER their aggregate band, never beyond the portfolio-level
// buyNeed. Buys are tax-DIRECTED: (account, under-group) cells are filled in
// descending _placementScore order, so each under-band group is bought first in
// the account type that holds it most tax-efficiently — cash isolation intact
// (an account never buys more than it holds in cash). Eligibility (canHold) gates
// every buy; a group with remaining need but no eligible candidate in an account
// surfaces as that account's unmet group (existing reporting preserved).
func (s *RebalanceService) buildBuyRecs(
	accounts []models.Account,
	positions []accountPosition,
	buyNeedByGroup map[string]decimal.Decimal,
	sellCashByAccount map[uuidx.UUID]decimal.Decimal,
	snapshots map[string]*tickerSnapshot,
	aggByGroup map[string]groupAgg,
	accountTypeByID map[uuidx.UUID]*string,
	availableTypes map[string]bool,
	restrictOverseas bool,
) (
	[]models.RebalanceRecommendation,
	map[uuidx.UUID]decimal.Decimal,
	map[uuidx.UUID][]string,
	map[uuidx.UUID][]models.RebalanceRecommendation,
) {
	var recs []models.RebalanceRecommendation
	recsByAccountID := map[uuidx.UUID][]models.RebalanceRecommendation{}

	cashByAccount := map[uuidx.UUID]decimal.Decimal{}
	for _, a := range accounts {
		cashByAccount[a.ID] = a.CashBalance.Decimal.Add(sellCashByAccount[a.ID])
	}
	remaining := map[string]decimal.Decimal{}
	for g, v := range buyNeedByGroup {
		remaining[g] = v
	}

	// (account, under-group) cells, filled in descending tax-placement score so the
	// most tax-appropriate account buys each group first.
	type cell struct {
		acctID uuidx.UUID
		group  string
		score  int
		gidx   int
	}
	gidxByGroup := map[string]int{}
	for i, g := range _groupOrder {
		gidxByGroup[g] = i
	}
	var cells []cell
	for _, a := range accounts {
		at := accountTypeByID[a.ID]
		for _, g := range _groupOrder {
			if !remaining[g].IsPositive() {
				continue
			}
			sc := -1
			if at != nil {
				sc = _placementScore[g][*at]
			}
			cells = append(cells, cell{a.ID, g, sc, gidxByGroup[g]})
		}
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].score != cells[j].score {
			return cells[i].score > cells[j].score
		}
		if cells[i].gidx != cells[j].gidx {
			return cells[i].gidx < cells[j].gidx
		}
		return cells[i].acctID.String() < cells[j].acctID.String()
	})

	accountByID := map[uuidx.UUID]models.Account{}
	for _, a := range accounts {
		accountByID[a.ID] = a
	}
	// couldNotBuy[acct][group] = the account had cash but no eligible candidate.
	couldNotBuy := map[uuidx.UUID]map[string]bool{}

	for _, c := range cells {
		need := remaining[c.group]
		if !need.IsPositive() {
			continue
		}
		cash := cashByAccount[c.acctID]
		if !cash.IsPositive() {
			continue
		}
		account := accountByID[c.acctID]
		candidate := s.selectBuyCandidateAccountScoped(c.acctID, c.group, positions, restrictOverseas, account.AccountType)
		if candidate == nil {
			candidate = s.selectBuyCandidatePortfolioFallback(c.group, snapshots, restrictOverseas, account.AccountType)
		}
		if candidate == nil {
			if couldNotBuy[c.acctID] == nil {
				couldNotBuy[c.acctID] = map[string]bool{}
			}
			couldNotBuy[c.acctID][c.group] = true
			continue
		}

		buyKRW := decimal.Min(cash, need)
		if !buyKRW.IsPositive() {
			continue
		}
		amountLocal := krwToLocal(buyKRW, candidate.currency, candidate.valueLocalBase, candidate.valueKRWBase)
		qty := calcQuantity(amountLocal, candidate.valueLocalBase, candidate.qtyBase)

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
			RebalanceGroupName: c.group,
			Reason:             buyReason(aggByGroup[c.group], c.group, account.AccountType, availableTypes),
			TriggerType:        "group",
			AmountKRW:          numeric.Wrap(buyKRW),
			AmountLocal:        numeric.Wrap(amountLocal),
		}
		recs = append(recs, rec)
		recsByAccountID[c.acctID] = append(recsByAccountID[c.acctID], rec)
		cashByAccount[c.acctID] = cash.Sub(buyKRW)
		remaining[c.group] = need.Sub(buyKRW)
	}

	// Unused cash + unmet groups per account. A group is unmet for an account that
	// still has cash but could not buy it (ineligible / no candidate) while the
	// group still needs portfolio-level buying.
	unusedCashByAccount := map[uuidx.UUID]decimal.Decimal{}
	unmetByAccount := map[uuidx.UUID][]string{}
	for _, a := range accounts {
		unusedCashByAccount[a.ID] = cashByAccount[a.ID]
		var unmet []string
		if cashByAccount[a.ID].IsPositive() {
			for _, g := range _groupOrder {
				if remaining[g].IsPositive() && couldNotBuy[a.ID][g] {
					unmet = append(unmet, g)
				}
			}
		}
		unmetByAccount[a.ID] = unmet
	}
	return recs, unusedCashByAccount, unmetByAccount, recsByAccountID
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

// --- recommendation reasons ---
//
// Reasons are framed at the AGGREGATE level: a group trades because its
// portfolio-wide weight breached its band. The account is named only to explain
// WHERE the necessary trade lands (tax-direction), never implying a per-account
// target.

func sellReason(agg groupAgg, gname string, accountType *string, availableTypes map[string]bool) string {
	return fmt.Sprintf(
		"합산 비중 초과 — %s 현재 %.2f%% > 목표 %.2f%%(±%.0f%%). 세금 영향이 작은 계좌부터 목표까지 감축, 이 계좌(%s)에서 매도 (세금 선호: %s)",
		gname, floatOf(agg.currentPct), floatOf(agg.targetPct), floatOf(agg.bandPct),
		accountTypeLabel(accountType), preferredAccountTypesLabel(gname, availableTypes))
}

func buyReason(agg groupAgg, gname string, accountType *string, availableTypes map[string]bool) string {
	return fmt.Sprintf(
		"합산 비중 부족 — %s 현재 %.2f%% < 목표 %.2f%%(±%.0f%%). 세금 효율이 높은 이 계좌(%s)에 우선 매수 (세금 선호: %s)",
		gname, floatOf(agg.currentPct), floatOf(agg.targetPct), floatOf(agg.bandPct),
		accountTypeLabel(accountType), preferredAccountTypesLabel(gname, availableTypes))
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
