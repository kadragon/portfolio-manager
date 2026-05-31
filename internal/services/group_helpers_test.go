package services

import (
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/shopspring/decimal"
)

func TestToGroup(t *testing.T) {
	// valid name (with surrounding whitespace stripped)
	if got, ok := toGroup("  국내성장 "); !ok || got != "국내성장" {
		t.Errorf("toGroup(국내성장) = %q,%v, want 국내성장,true", got, ok)
	}
	if got, ok := toGroup("해외배당"); !ok || got != "해외배당" {
		t.Errorf("toGroup(해외배당) = %q,%v, want 해외배당,true", got, ok)
	}
	// invalid name
	if _, ok := toGroup("미분류"); ok {
		t.Errorf("toGroup(미분류) should be ok=false")
	}
}

func TestBuildTargetByGroup(t *testing.T) {
	s := NewRebalanceService()
	groups := []models.Group{
		{Name: "국내성장", TargetPercentage: 60},
		{Name: "해외배당", TargetPercentage: 40},
		{Name: "미분류", TargetPercentage: 99}, // unmapped — ignored
	}
	got := s.buildTargetByGroup(groups)
	// result always carries every group-order key (5), zero-initialized.
	if len(got) != 5 {
		t.Fatalf("buildTargetByGroup len = %d, want 5", len(got))
	}
	if !got["국내성장"].Equal(decimal.NewFromInt(60)) {
		t.Errorf("국내성장 = %s, want 60", got["국내성장"].String())
	}
	if !got["해외배당"].Equal(decimal.NewFromInt(40)) {
		t.Errorf("해외배당 = %s, want 40", got["해외배당"].String())
	}
	// unmapped contributes nothing; an untouched valid group stays zero.
	if !got["국내배당"].IsZero() {
		t.Errorf("국내배당 = %s, want 0", got["국내배당"].String())
	}
}
