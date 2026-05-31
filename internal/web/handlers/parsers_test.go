package handlers

import (
	"testing"

	"github.com/kadragon/portfolio-manager/internal/numeric"
)

func TestParseQuantity(t *testing.T) {
	if got := parseQuantity("10"); !got.Equal(numeric.FromInt(10).Decimal) {
		t.Errorf("parseQuantity(10) = %s, want 10", got.String())
	}
	if got := parseQuantity("2.5"); got.String() != "2.5" {
		t.Errorf("parseQuantity(2.5) = %s, want 2.5", got.String())
	}
	// invalid → zero
	if got := parseQuantity("abc"); !got.IsZero() {
		t.Errorf("parseQuantity(abc) = %s, want 0", got.String())
	}
	if got := parseQuantity(""); !got.IsZero() {
		t.Errorf("parseQuantity(empty) = %s, want 0", got.String())
	}
}

func TestParseDepositAmount(t *testing.T) {
	got, err := parseDepositAmount("1000000")
	if err != nil {
		t.Fatalf("parseDepositAmount(1000000) error: %v", err)
	}
	if !got.Equal(numeric.FromInt(1000000).Decimal) {
		t.Errorf("parseDepositAmount(1000000) = %s, want 1000000", got.String())
	}
	if _, err := parseDepositAmount("not-a-number"); err == nil {
		t.Errorf("parseDepositAmount(invalid) should error")
	}
}

func TestNormalizeBulkError(t *testing.T) {
	if got := normalizeBulkError("quantity must be greater than zero"); got != "모든 수량은 0보다 커야 합니다." {
		t.Errorf("known message not mapped: %q", got)
	}
	if got := normalizeBulkError(""); got != "보유 수량 일괄 저장 중 오류가 발생했습니다." {
		t.Errorf("empty message default = %q", got)
	}
	if got := normalizeBulkError("some other error"); got != "some other error" {
		t.Errorf("unknown message should pass through, got %q", got)
	}
}

func TestNormalizeKisAccountNo(t *testing.T) {
	// "12345678-01" → 10 digits split 8+2
	cano, prdt, err := normalizeKisAccountNo("12345678-01")
	if err != nil || cano != "12345678" || prdt != "01" {
		t.Errorf("normalizeKisAccountNo(12345678-01) = %q,%q,%v", cano, prdt, err)
	}
	// bare 10 digits
	cano2, prdt2, err2 := normalizeKisAccountNo("1234567890")
	if err2 != nil || cano2 != "12345678" || prdt2 != "90" {
		t.Errorf("normalizeKisAccountNo(1234567890) = %q,%q,%v", cano2, prdt2, err2)
	}
	// not exactly 10 digits → error
	if _, _, err := normalizeKisAccountNo("12345"); err == nil {
		t.Errorf("normalizeKisAccountNo(<10 digits) should error")
	}
	if _, _, err := normalizeKisAccountNo("1234567890123"); err == nil {
		t.Errorf("normalizeKisAccountNo(>10 digits) should error")
	}
}
