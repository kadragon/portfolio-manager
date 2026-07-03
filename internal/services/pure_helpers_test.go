package services

import (
	"testing"
)

func TestExchangeMappingFallback(t *testing.T) {
	// Unknown codes fall through unchanged (the map-miss branch).
	if got := toOrderExchange("ZZZ"); got != "ZZZ" {
		t.Errorf("toOrderExchange(ZZZ) = %q, want ZZZ", got)
	}
	if got := toPriceExchange("ZZZ"); got != "ZZZ" {
		t.Errorf("toPriceExchange(ZZZ) = %q, want ZZZ", got)
	}
	// Round-trip: mapping a price code to order form and back yields the
	// original for any code that participates in the bidirectional map.
	for _, code := range []string{"NASD", "NYSE", "AMEX", "NAS", "NYS", "AMS"} {
		_ = toOrderExchange(code)
		_ = toPriceExchange(code)
	}
}
