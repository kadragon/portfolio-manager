package templates

import (
	"testing"

	"github.com/kadragon/portfolio-manager/internal/numeric"
)

// TestFormatQty pins the quantity-formatting rule: overseas tickers (len < 5)
// show one decimal place; domestic tickers (len >= 5) show an integer with
// thousands separators.
func TestFormatQty(t *testing.T) {
	cases := []struct {
		name, ticker, qty, want string
	}{
		{"domestic thousands", "005930", "1234", "1,234"},
		{"domestic large", "360750", "1234567", "1,234,567"},
		{"domestic small", "458730", "12", "12"},
		{"overseas one decimal", "AAPL", "10.5", "10.5"},
		{"overseas integer gets decimal", "VYM", "7", "7.0"},
		{"five-char ticker is domestic", "GOOGL", "1000", "1,000"},
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
