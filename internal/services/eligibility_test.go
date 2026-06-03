package services

import "testing"

func TestCanHold(t *testing.T) {
	brokerage := "brokerage"
	irp := "irp"
	pension := "pension"
	isa := "isa"
	bogus := "margin"

	cases := []struct {
		name        string
		accountType *string
		ticker      string
		isETF       bool
		want        bool
	}{
		// brokerage: anything
		{"brokerage domestic etf", &brokerage, "069500", true, true},
		{"brokerage us etf", &brokerage, "SCHD", true, true},
		{"brokerage domestic stock", &brokerage, "005930", false, true},
		{"brokerage us stock", &brokerage, "AAPL", false, true},

		// IRP: domestic-listed ETF only
		{"irp domestic etf", &irp, "069500", true, true},
		{"irp us etf", &irp, "SCHD", true, false},
		{"irp domestic individual stock", &irp, "005930", false, false},
		// key case: 해외성장 domestic-listed ETF (TIGER 미국S&P500) IS irp-eligible —
		// eligibility keys off listing+class, not the 국내/해외 group label.
		{"irp foreign-tracking domestic etf", &irp, "360750", true, true},

		// pension: same rules as IRP
		{"pension domestic etf", &pension, "069500", true, true},
		{"pension us etf", &pension, "QQQ", true, false},
		{"pension domestic stock", &pension, "000660", false, false},

		// ISA: domestic-listed, ETF or stock
		{"isa domestic etf", &isa, "069500", true, true},
		{"isa domestic stock", &isa, "005930", false, true},
		{"isa us etf", &isa, "SCHD", true, false},
		{"isa us stock", &isa, "AAPL", false, false},

		// nil / unknown → strict block
		{"nil account type", nil, "069500", true, false},
		{"unknown account type", &bogus, "069500", true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canHold(tc.accountType, tc.ticker, tc.isETF); got != tc.want {
				t.Errorf("canHold(%v, %q, etf=%v) = %v, want %v", tc.accountType, tc.ticker, tc.isETF, got, tc.want)
			}
		})
	}
}
