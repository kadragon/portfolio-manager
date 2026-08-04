package models

import "testing"

func TestValidSecurityGroup(t *testing.T) {
	valid := []string{"ST", "EF", "EN", "EW", "MF", "RT", "FE", "FS"}
	for _, code := range valid {
		if !ValidSecurityGroup(code) {
			t.Errorf("ValidSecurityGroup(%q) = false, want true", code)
		}
	}
	// empty is NOT a recognized code (handler accepts it separately to clear the field)
	if ValidSecurityGroup("") {
		t.Errorf("ValidSecurityGroup(\"\") = true, want false")
	}
	invalid := []string{"ef", "st", "XX", "ETF", "STOCK", " ST", "ST "}
	for _, code := range invalid {
		if ValidSecurityGroup(code) {
			t.Errorf("ValidSecurityGroup(%q) = true, want false", code)
		}
	}
}

func TestWellFormedSecurityGroup(t *testing.T) {
	// known codes and codes KIS could add later are both well formed
	wellFormed := []string{"ST", "EF", "XX", "QQ"}
	for _, code := range wellFormed {
		if !WellFormedSecurityGroup(code) {
			t.Errorf("WellFormedSecurityGroup(%q) = false, want true", code)
		}
	}
	malformed := []string{"", "S", "STO", "st", "sT", "S1", "1A", "S ", " S", "S-"}
	for _, code := range malformed {
		if WellFormedSecurityGroup(code) {
			t.Errorf("WellFormedSecurityGroup(%q) = true, want false", code)
		}
	}
}

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
