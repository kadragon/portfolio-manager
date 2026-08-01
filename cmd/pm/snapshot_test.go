package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/numeric"
)

func newSnapshotContainer(t *testing.T) *container.Container {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return container.NewWithQueries(sqlDB, q)
}

func mustDec(t *testing.T, s string) numeric.Decimal {
	t.Helper()
	d, err := numeric.FromString(s)
	if err != nil {
		t.Fatalf("FromString(%q): %v", s, err)
	}
	return d
}

// seedSnapshot builds a two-group portfolio: 국내(KRW) with one holding, and
// 해외(USD) with one holding priced today, plus a target-only 채권 group with no
// holdings. Returns the container ready for buildSnapshot.
func seedSnapshot(t *testing.T, c *container.Container) {
	t.Helper()
	ctx := context.Background()
	today := datex.FromTime(ktime.NowKST())

	gKR, err := c.Groups.Create(ctx, "국내", 60)
	if err != nil {
		t.Fatalf("group 국내: %v", err)
	}
	gUS, err := c.Groups.Create(ctx, "해외", 30)
	if err != nil {
		t.Fatalf("group 해외: %v", err)
	}
	if _, err := c.Groups.Create(ctx, "채권", 10); err != nil { // target-only, no holdings
		t.Fatalf("group 채권: %v", err)
	}

	acc, err := c.Accounts.Create(ctx, "ISA", mustDec(t, "50000"))
	if err != nil {
		t.Fatalf("account: %v", err)
	}

	sKR, err := c.Stocks.Create(ctx, "005930", gKR.ID)
	if err != nil {
		t.Fatalf("stock KR: %v", err)
	}
	sUS, err := c.Stocks.Create(ctx, "VOO", gUS.ID)
	if err != nil {
		t.Fatalf("stock US: %v", err)
	}

	krx := sql.NullString{String: "KRX", Valid: true}
	amex := sql.NullString{String: "AMEX", Valid: true}
	if _, err := c.StockPrices.Save(ctx, "005930", today, mustDec(t, "70000"), "KRW", "삼성전자", krx); err != nil {
		t.Fatalf("price KR: %v", err)
	}
	if _, err := c.StockPrices.Save(ctx, "VOO", today, mustDec(t, "500"), "USD", "Vanguard S&P 500", amex); err != nil {
		t.Fatalf("price US: %v", err)
	}

	if _, err := c.Holdings.Create(ctx, acc.ID, sKR.ID, mustDec(t, "10")); err != nil {
		t.Fatalf("holding KR: %v", err)
	}
	if _, err := c.Holdings.Create(ctx, acc.ID, sUS.ID, mustDec(t, "2")); err != nil {
		t.Fatalf("holding US: %v", err)
	}
}

func TestBuildSnapshotValuesAndWeights(t *testing.T) {
	ctx := context.Background()
	c := newSnapshotContainer(t)
	seedSnapshot(t, c)

	out, err := buildSnapshot(ctx, c, 1400, 7)
	if err != nil {
		t.Fatalf("buildSnapshot: %v", err)
	}

	// 국내: 10 * 70000 = 700_000. 해외: 2 * 500 * 1400 = 1_400_000. cash 50_000.
	const wantHoldings = 2_100_000
	const wantCash = 50_000
	if out.TotalHoldingsKRW != wantHoldings {
		t.Errorf("TotalHoldingsKRW = %d, want %d", out.TotalHoldingsKRW, wantHoldings)
	}
	if out.TotalCashKRW != wantCash {
		t.Errorf("TotalCashKRW = %d, want %d", out.TotalCashKRW, wantCash)
	}
	if out.TotalKRW != wantHoldings+wantCash {
		t.Errorf("TotalKRW = %d, want %d", out.TotalKRW, wantHoldings+wantCash)
	}
	if out.FxUSDKRW != 1400 {
		t.Errorf("FxUSDKRW = %v, want 1400", out.FxUSDKRW)
	}

	byGroup := map[string]snapshotGroup{}
	for _, g := range out.Groups {
		byGroup[g.Name] = g
	}
	// USD group values larger and must sort first.
	if out.Groups[0].Name != "해외" {
		t.Errorf("first group = %q, want 해외 (largest value first)", out.Groups[0].Name)
	}
	if got := byGroup["해외"].ValueKRW; got != 1_400_000 {
		t.Errorf("해외 value = %d, want 1400000", got)
	}
	// weight_pct is over total including cash: 1_400_000 / 2_150_000 = 65.12%.
	if w := byGroup["해외"].WeightPct; w == nil || *w < 65.11 || *w > 65.13 {
		t.Errorf("해외 weight = %v, want ~65.12", w)
	}
	// Target-only 채권 must still appear with a nil weight != nil value 0.
	bond, ok := byGroup["채권"]
	if !ok {
		t.Fatal("채권 group missing from snapshot")
	}
	if bond.ValueKRW != 0 {
		t.Errorf("채권 value = %d, want 0", bond.ValueKRW)
	}
	if bond.DBTargetPct == nil || *bond.DBTargetPct != 10 {
		t.Errorf("채권 db_target_pct = %v, want 10", bond.DBTargetPct)
	}
	if len(out.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", out.Warnings)
	}
}

func TestBuildSnapshotUsesBankersRoundingForValues(t *testing.T) {
	ctx := context.Background()
	c := newSnapshotContainer(t)
	group, err := c.Groups.Create(ctx, "half", 100)
	if err != nil {
		t.Fatalf("group: %v", err)
	}
	account, err := c.Accounts.Create(ctx, "half account", numeric.Zero)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	stock, err := c.Stocks.Create(ctx, "HALF", group.ID)
	if err != nil {
		t.Fatalf("stock: %v", err)
	}
	today := datex.FromTime(ktime.NowKST())
	if _, err := c.StockPrices.Save(ctx, "HALF", today, mustDec(t, "2.5"), "KRW", "Half", sql.NullString{}); err != nil {
		t.Fatalf("price: %v", err)
	}
	if _, err := c.Holdings.Create(ctx, account.ID, stock.ID, numeric.FromInt(1)); err != nil {
		t.Fatalf("holding: %v", err)
	}

	out, err := buildSnapshot(ctx, c, 1400, 7)
	if err != nil {
		t.Fatalf("buildSnapshot: %v", err)
	}
	if got := out.Accounts[0].Holdings[0].ValueKRW; got != 2 {
		t.Errorf("holding value = %d, want banker-rounded 2", got)
	}
	if out.TotalHoldingsKRW != 2 || out.TotalKRW != 2 {
		t.Errorf("totals = (%d, %d), want (2, 2)", out.TotalHoldingsKRW, out.TotalKRW)
	}
	if got := out.Groups[0].ValueKRW; got != 2 {
		t.Errorf("group value = %d, want banker-rounded 2", got)
	}
}

func TestBuildSnapshotUsesBankersRoundingForWeightBoundary(t *testing.T) {
	ctx := context.Background()
	c := newSnapshotContainer(t)
	weightGroup, err := c.Groups.Create(ctx, "weight", 50)
	if err != nil {
		t.Fatalf("weight group: %v", err)
	}
	otherGroup, err := c.Groups.Create(ctx, "other", 50)
	if err != nil {
		t.Fatalf("other group: %v", err)
	}
	account, err := c.Accounts.Create(ctx, "weight account", mustDec(t, "85.155"))
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	weightStock, err := c.Stocks.Create(ctx, "WEIGHT", weightGroup.ID)
	if err != nil {
		t.Fatalf("weight stock: %v", err)
	}
	otherStock, err := c.Stocks.Create(ctx, "OTHER", otherGroup.ID)
	if err != nil {
		t.Fatalf("other stock: %v", err)
	}
	today := datex.FromTime(ktime.NowKST())
	for ticker, price := range map[string]string{"WEIGHT": "12.345", "OTHER": "2.5"} {
		if _, err := c.StockPrices.Save(ctx, ticker, today, mustDec(t, price), "KRW", ticker, sql.NullString{}); err != nil {
			t.Fatalf("price %s: %v", ticker, err)
		}
	}
	if _, err := c.Holdings.Create(ctx, account.ID, weightStock.ID, numeric.FromInt(1)); err != nil {
		t.Fatalf("weight holding: %v", err)
	}
	if _, err := c.Holdings.Create(ctx, account.ID, otherStock.ID, numeric.FromInt(1)); err != nil {
		t.Fatalf("other holding: %v", err)
	}

	out, err := buildSnapshot(ctx, c, 1400, 7)
	if err != nil {
		t.Fatalf("buildSnapshot: %v", err)
	}
	var weight *float64
	for _, group := range out.Groups {
		if group.Name == "weight" {
			weight = group.WeightPct
			break
		}
	}
	if weight == nil || *weight != 12.34 {
		t.Errorf("weight = %v, want banker-rounded 12.34", weight)
	}
}

func TestBuildSnapshotMissingAndStalePriceWarnings(t *testing.T) {
	ctx := context.Background()
	c := newSnapshotContainer(t)

	g, err := c.Groups.Create(ctx, "국내", 100)
	if err != nil {
		t.Fatalf("group: %v", err)
	}
	acc, err := c.Accounts.Create(ctx, "ISA", numeric.Zero)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	// NOPRICE: no price row at all → value 0 + warning.
	sNo, err := c.Stocks.Create(ctx, "NOPRICE", g.ID)
	if err != nil {
		t.Fatalf("stock NOPRICE: %v", err)
	}
	// STALE: price 30 days old → stale warning.
	sStale, err := c.Stocks.Create(ctx, "STALE", g.ID)
	if err != nil {
		t.Fatalf("stock STALE: %v", err)
	}
	old := datex.FromTime(ktime.NowKST().Add(-30 * 24 * time.Hour))
	krx := sql.NullString{String: "KRX", Valid: true}
	if _, err := c.StockPrices.Save(ctx, "STALE", old, mustDec(t, "1000"), "KRW", "Stale Co", krx); err != nil {
		t.Fatalf("price STALE: %v", err)
	}

	if _, err := c.Holdings.Create(ctx, acc.ID, sNo.ID, mustDec(t, "5")); err != nil {
		t.Fatalf("holding NOPRICE: %v", err)
	}
	if _, err := c.Holdings.Create(ctx, acc.ID, sStale.ID, mustDec(t, "5")); err != nil {
		t.Fatalf("holding STALE: %v", err)
	}

	out, err := buildSnapshot(ctx, c, 1400, 7)
	if err != nil {
		t.Fatalf("buildSnapshot: %v", err)
	}

	var sawNoPrice, sawStale bool
	for _, w := range out.Warnings {
		if w == "no price for NOPRICE — value set to 0" {
			sawNoPrice = true
		}
		if w == "price for STALE is 30 days old ("+old.ISO()+")" {
			sawStale = true
		}
	}
	if !sawNoPrice {
		t.Errorf("missing no-price warning; warnings = %v", out.Warnings)
	}
	if !sawStale {
		t.Errorf("missing stale warning; warnings = %v", out.Warnings)
	}
	// STALE still valued (30 days old is stale but priced): 5 * 1000 = 5000.
	if out.TotalHoldingsKRW != 5000 {
		t.Errorf("TotalHoldingsKRW = %d, want 5000", out.TotalHoldingsKRW)
	}
}

// TestBuildSnapshotEmptyDBEmitsEmptyArrays guards the JSON shape: an empty DB
// must serialize accounts/groups as [] (not null), matching the former
// snapshot.py so downstream consumers can iterate them unconditionally.
func TestBuildSnapshotEmptyDBEmitsEmptyArrays(t *testing.T) {
	ctx := context.Background()
	c := newSnapshotContainer(t)

	out, err := buildSnapshot(ctx, c, 1400, 7)
	if err != nil {
		t.Fatalf("buildSnapshot: %v", err)
	}
	blob, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"accounts":[]`, `"groups":[]`, `"warnings":[]`} {
		if !strings.Contains(string(blob), want) {
			t.Errorf("empty-DB snapshot missing %s; got %s", want, blob)
		}
	}
}

// TestBuildSnapshotGroupOrderDeterministicForEqualValues guards against the
// map-iteration nondeterminism a review caught: groups tied on value (here two
// target-only groups at 0) must order by name every run, not flap.
func TestBuildSnapshotGroupOrderDeterministicForEqualValues(t *testing.T) {
	ctx := context.Background()
	c := newSnapshotContainer(t)
	for _, name := range []string{"채권", "금", "현금"} { // all target-only → value 0
		if _, err := c.Groups.Create(ctx, name, 10); err != nil {
			t.Fatalf("group %s: %v", name, err)
		}
	}

	out, err := buildSnapshot(ctx, c, 1400, 7)
	if err != nil {
		t.Fatalf("buildSnapshot: %v", err)
	}
	got := []string{out.Groups[0].Name, out.Groups[1].Name, out.Groups[2].Name}
	// All three tie at value 0, so they must come out strictly name-sorted.
	if got[0] > got[1] || got[1] > got[2] {
		t.Errorf("equal-value groups not name-ordered: %v", got)
	}
}
