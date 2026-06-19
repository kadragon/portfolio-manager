package models

import "testing"

func TestValidAssetClass(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"etf", true},
		{"stock", true},
		{"", false},
		{"unknown", false},
		{"ETF", false},
		{"fund", false},
	}
	for _, tc := range cases {
		if got := ValidAssetClass(tc.in); got != tc.want {
			t.Errorf("ValidAssetClass(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
