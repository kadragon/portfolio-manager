package templates

import (
	"testing"

	"github.com/kadragon/portfolio-manager/internal/numeric"
)

// TestFormatQty pins the quantity-formatting rule: domestic tickers are exactly
// 6-character KRX codes (cf. kis.IsDomesticTicker) and show an integer with
// thousands separators; every other ticker is overseas and shows one decimal
// place so fractional shares are not lost.
func TestFormatQty(t *testing.T) {
	cases := []struct {
		name, ticker, qty, want string
	}{
		{"domestic thousands", "005930", "1234", "1,234"},
		{"domestic large", "360750", "1234567", "1,234,567"},
		{"domestic small", "458730", "12", "12"},
		{"overseas one decimal", "AAPL", "10.5", "10.5"},
		{"overseas integer gets decimal", "VYM", "7", "7.0"},
		{"five-char overseas keeps fraction", "GOOGL", "10.5", "10.5"},
		{"five-char overseas integer", "GOOGL", "1000", "1000.0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q, err := numeric.FromString(c.qty)
			if err != nil {
				t.Fatalf("parse %q: %v", c.qty, err)
			}
			if got := formatQty(c.ticker, q); got != c.want {
				t.Errorf("formatQty(%q, %q) = %q, want %q", c.ticker, c.qty, got, c.want)
			}
		})
	}
}

// TestFormatRate pins the exchange-rate display rule: nil or non-positive rate
// renders "-", a positive rate renders as a KRW-formatted string.
func TestFormatRate(t *testing.T) {
	t.Run("nil is dash", func(t *testing.T) {
		if got := formatRate(nil); got != "-" {
			t.Errorf("formatRate(nil) = %q, want %q", got, "-")
		}
	})
	t.Run("zero is dash", func(t *testing.T) {
		zero := numeric.Zero
		if got := formatRate(&zero); got != "-" {
			t.Errorf("formatRate(0) = %q, want %q", got, "-")
		}
	})
	t.Run("positive is KRW", func(t *testing.T) {
		rate, err := numeric.FromString("1350")
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if got := formatRate(&rate); got != "₩1,350" {
			t.Errorf("formatRate(1350) = %q, want %q", got, "₩1,350")
		}
	})
}
