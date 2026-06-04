package stockformat_test

import (
	"testing"

	"github.com/kadragon/portfolio-manager/internal/stockformat"
)

func TestFormatName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"KODEX 200증권상장지수투자신탁(주식)", "KODEX 200"},
		{"삼성전자", "삼성전자"},
		{"", ""},
		{"증권상장지수투자신탁(주식)", ""},
		{"미래에셋 TIGER MSCI Korea Total Return증권상장지수투자신탁(", "미래에셋 TIGER MSCI Korea Total Return"},
		{"TIGER MSCI Korea증권상장지수투자신탁(주식-파생형)", "TIGER MSCI Korea"},
	}
	for _, c := range cases {
		got := stockformat.FormatName(c.in)
		if got != c.want {
			t.Errorf("FormatName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
