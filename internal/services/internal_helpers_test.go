package services

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestDomKeyAndSellDomKey(t *testing.T) {
	const domestic = "005930" // 6 digits
	const overseas = "AAPL"
	if got := domKey(domestic); got != 0 {
		t.Errorf("domKey(domestic) = %d, want 0", got)
	}
	if got := domKey(overseas); got != 1 {
		t.Errorf("domKey(overseas) = %d, want 1", got)
	}
	if got := sellDomKey(domestic); got != 1 {
		t.Errorf("sellDomKey(domestic) = %d, want 1", got)
	}
	if got := sellDomKey(overseas); got != 0 {
		t.Errorf("sellDomKey(overseas) = %d, want 0", got)
	}
}

func TestIsDomesticTickerHelper(t *testing.T) {
	if !isDomesticTicker("005930") {
		t.Errorf("6-digit ticker should be domestic")
	}
	if isDomesticTicker("AAPL") {
		t.Errorf("non-6-char ticker should be overseas")
	}
	if isDomesticTicker("00593") {
		t.Errorf("5-digit ticker should not be domestic")
	}
}

func TestCalcQuantity(t *testing.T) {
	amount := decimal.NewFromInt(1000)
	posValue := decimal.NewFromInt(500)
	posQty := decimal.NewFromInt(10)
	got := calcQuantity(amount, posValue, posQty)
	if got == nil {
		t.Fatalf("calcQuantity returned nil for valid inputs")
	}
	// 1000/500 * 10 = 20
	if !got.Equal(decimal.NewFromInt(20)) {
		t.Errorf("calcQuantity = %s, want 20", got.String())
	}

	if calcQuantity(amount, decimal.Zero, posQty) != nil {
		t.Errorf("calcQuantity with zero posValue should be nil")
	}
	if calcQuantity(amount, posValue, decimal.Zero) != nil {
		t.Errorf("calcQuantity with zero posQty should be nil")
	}
}

func TestPtrDecimal(t *testing.T) {
	if ptrDecimal(nil) != nil {
		t.Errorf("ptrDecimal(nil) should be nil")
	}
	d := decimal.NewFromInt(7)
	got := ptrDecimal(&d)
	if got == nil || !got.Decimal.Equal(d) {
		t.Errorf("ptrDecimal(&7) = %v, want 7", got)
	}
}

func TestFloatOf(t *testing.T) {
	if got := floatOf(decimal.NewFromFloat(3.5)); got != 3.5 {
		t.Errorf("floatOf(3.5) = %v, want 3.5", got)
	}
	if got := floatOf(decimal.Zero); got != 0 {
		t.Errorf("floatOf(0) = %v, want 0", got)
	}
}

func TestFilterPositionsAndSumValueKRW(t *testing.T) {
	positions := []accountPosition{
		{ticker: "005930", valueKRW: decimal.NewFromInt(100)},
		{ticker: "AAPL", valueKRW: decimal.NewFromInt(200)},
		{ticker: "000660", valueKRW: decimal.NewFromInt(300)},
	}
	domestic := filterPositions(positions, func(p accountPosition) bool {
		return isDomesticTicker(p.ticker)
	})
	if len(domestic) != 2 {
		t.Fatalf("filterPositions domestic = %d, want 2", len(domestic))
	}
	if got := sumValueKRW(domestic); !got.Equal(decimal.NewFromInt(400)) {
		t.Errorf("sumValueKRW(domestic) = %s, want 400", got.String())
	}
	if got := sumValueKRW(nil); !got.Equal(decimal.Zero) {
		t.Errorf("sumValueKRW(nil) = %s, want 0", got.String())
	}
}

func TestKrwToLocal(t *testing.T) {
	amountKRW := decimal.NewFromInt(13500)
	// USD with fx = valueKRW/valueLocal = 13500/100 = 135 → 13500/135 = 100
	got := krwToLocal(amountKRW, "USD", decimal.NewFromInt(100), decimal.NewFromInt(13500))
	if !got.Equal(decimal.NewFromInt(100)) {
		t.Errorf("krwToLocal(USD) = %s, want 100", got.String())
	}
	// KRW currency → unchanged
	if got := krwToLocal(amountKRW, "KRW", decimal.NewFromInt(100), decimal.NewFromInt(13500)); !got.Equal(amountKRW) {
		t.Errorf("krwToLocal(KRW) = %s, want 13500", got.String())
	}
	// USD with non-positive valueLocal → unchanged
	if got := krwToLocal(amountKRW, "USD", decimal.Zero, decimal.NewFromInt(13500)); !got.Equal(amountKRW) {
		t.Errorf("krwToLocal(USD, zero local) = %s, want 13500", got.String())
	}
}

func TestShiftYearsAndMonths(t *testing.T) {
	base := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)
	if got := shiftYears(base, 1); got.Year() != 2025 || got.Month() != time.March || got.Day() != 15 {
		t.Errorf("shiftYears(-1) = %v, want 2025-03-15", got)
	}
	if got := shiftMonths(base, 2); got.Year() != 2026 || got.Month() != time.January || got.Day() != 15 {
		t.Errorf("shiftMonths(-2) = %v, want 2026-01-15", got)
	}
}

func TestPrevBizDay(t *testing.T) {
	// Saturday → Friday (-1 day)
	sat := time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC)
	if got := prevBizDay(sat).Weekday(); got != time.Friday {
		t.Errorf("prevBizDay(Sat) weekday = %v, want Friday", got)
	}
	// Sunday → Friday (-2 days)
	sun := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)
	if got := prevBizDay(sun).Weekday(); got != time.Friday {
		t.Errorf("prevBizDay(Sun) weekday = %v, want Friday", got)
	}
	// Weekday → unchanged
	wed := time.Date(2026, time.March, 18, 0, 0, 0, 0, time.UTC)
	if got := prevBizDay(wed); !got.Equal(wed) {
		t.Errorf("prevBizDay(Wed) = %v, want unchanged", got)
	}
}

func TestComputeTargetDates(t *testing.T) {
	today := time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC) // Monday
	dates := computeTargetDates(today)
	for _, label := range []string{"1y", "6m", "1m", "1d"} {
		d, ok := dates[label]
		if !ok {
			t.Errorf("computeTargetDates missing label %q", label)
			continue
		}
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			t.Errorf("computeTargetDates[%q] = %v lands on weekend %v", label, d, wd)
		}
	}
	// 1d before Monday is Sunday → prevBizDay → Friday
	if got := dates["1d"].Weekday(); got != time.Friday {
		t.Errorf("computeTargetDates[1d] weekday = %v, want Friday", got)
	}
}
