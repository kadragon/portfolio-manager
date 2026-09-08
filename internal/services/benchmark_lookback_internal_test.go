package services

import "testing"

// The walk-back may only reach back as far as a close is still allowed to stand
// in for a deposit: a candidate outside maxBenchmarkPriceGap would be saved by
// price sync and then rejected by staleBenchmarkPrice, costing a call to store a
// row nothing can use.
func TestBenchmarkDepositLookbackStaysInsidePriceGap(t *testing.T) {
	if benchmarkDepositLookback >= maxBenchmarkPriceGap {
		t.Errorf("benchmarkDepositLookback (%v) must stay under maxBenchmarkPriceGap (%v)",
			benchmarkDepositLookback, maxBenchmarkPriceGap)
	}
}
