package services

import (
	"testing"

	"github.com/kadragon/portfolio-manager/internal/uuidx"
	"github.com/shopspring/decimal"
)

func TestSelectBuyCandidateAccountScoped(t *testing.T) {
	s := NewRebalanceService()
	bk := "brokerage"
	acc := uuidx.New()
	other := uuidx.New()

	positions := []accountPosition{
		// wrong account — excluded
		{accountID: other, ticker: "005930", rebalanceGroup: "G", valueLocal: decimal.NewFromInt(10), valueKRW: decimal.NewFromInt(10)},
		// wrong group — excluded
		{accountID: acc, ticker: "000660", rebalanceGroup: "X", valueLocal: decimal.NewFromInt(10), valueKRW: decimal.NewFromInt(10)},
		// zero local value — excluded
		{accountID: acc, ticker: "035720", rebalanceGroup: "G", valueLocal: decimal.Zero, valueKRW: decimal.NewFromInt(10)},
		// eligible domestic, lower KRW
		{accountID: acc, ticker: "005380", rebalanceGroup: "G", valueLocal: decimal.NewFromInt(5), valueKRW: decimal.NewFromInt(100)},
		// eligible domestic, higher KRW → winner
		{accountID: acc, ticker: "068270", rebalanceGroup: "G", valueLocal: decimal.NewFromInt(5), valueKRW: decimal.NewFromInt(200)},
		// eligible overseas
		{accountID: acc, ticker: "AAPL", rebalanceGroup: "G", valueLocal: decimal.NewFromInt(5), valueKRW: decimal.NewFromInt(999)},
	}

	// restrictOverseas=false: domestic ranks before overseas (domKey), then highest KRW
	got := s.selectBuyCandidateAccountScoped(acc, "G", positions, false, &bk)
	if got == nil {
		t.Fatalf("expected a candidate, got nil")
	}
	if got.ticker != "068270" {
		t.Errorf("winner ticker = %q, want 068270 (domestic highest KRW)", got.ticker)
	}

	// restrictOverseas=true: overseas excluded, still 068270
	gotR := s.selectBuyCandidateAccountScoped(acc, "G", positions, true, &bk)
	if gotR == nil || gotR.ticker != "068270" {
		t.Errorf("restricted winner = %v, want 068270", gotR)
	}

	// no eligible positions → nil
	if c := s.selectBuyCandidateAccountScoped(uuidx.New(), "G", positions, false, &bk); c != nil {
		t.Errorf("unknown account should yield nil, got %v", c)
	}
}

func TestSelectBuyCandidatePortfolioFallback(t *testing.T) {
	s := NewRebalanceService()
	bk := "brokerage"

	snapshots := map[string]*tickerSnapshot{
		"005930": {ticker: "005930", rebalanceGroup: "G", totalValueLocal: decimal.NewFromInt(5), totalValueKRW: decimal.NewFromInt(100)},
		"068270": {ticker: "068270", rebalanceGroup: "G", totalValueLocal: decimal.NewFromInt(5), totalValueKRW: decimal.NewFromInt(300)},
		"AAPL":   {ticker: "AAPL", rebalanceGroup: "G", totalValueLocal: decimal.NewFromInt(5), totalValueKRW: decimal.NewFromInt(999)},
		"WRONG":  {ticker: "WRONG1", rebalanceGroup: "X", totalValueLocal: decimal.NewFromInt(5), totalValueKRW: decimal.NewFromInt(999)},
		"ZERO":   {ticker: "000660", rebalanceGroup: "G", totalValueLocal: decimal.Zero, totalValueKRW: decimal.NewFromInt(999)},
	}

	// domestic ranks first; highest KRW among domestic → 068270
	got := s.selectBuyCandidatePortfolioFallback("G", snapshots, false, &bk)
	if got == nil || got.ticker != "068270" {
		t.Errorf("portfolio fallback winner = %v, want 068270", got)
	}

	// restrictOverseas excludes AAPL — still 068270
	gotR := s.selectBuyCandidatePortfolioFallback("G", snapshots, true, &bk)
	if gotR == nil || gotR.ticker != "068270" {
		t.Errorf("restricted fallback = %v, want 068270", gotR)
	}

	// group with no eligible snapshots → nil
	if c := s.selectBuyCandidatePortfolioFallback("NONE", snapshots, false, &bk); c != nil {
		t.Errorf("empty group should yield nil, got %v", c)
	}
}
