package templates

import (
	"strings"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
)

func mustDec(t *testing.T, s string) numeric.Decimal {
	t.Helper()
	d, err := numeric.FromString(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return d
}

func row(t *testing.T, name, actual, target, diffVal string) models.GroupSummaryRow {
	return models.GroupSummaryRow{
		Group:     models.Group{Name: name},
		ActualPct: mustDec(t, actual),
		TargetPct: mustDec(t, target),
		DiffVal:   mustDec(t, diffVal),
	}
}

func TestDonutSVG(t *testing.T) {
	if got := donutSVG(nil); got != "" {
		t.Errorf("empty rows: want \"\", got %q", got)
	}

	rows := []models.GroupSummaryRow{
		row(t, "A", "42", "40", "1"),
		row(t, "B", "28", "30", "-1"),
		row(t, "C", "30", "30", "0"),
	}
	svg := donutSVG(rows)
	if !strings.HasPrefix(svg, "<svg") || !strings.HasSuffix(svg, "</svg>") {
		t.Fatalf("not a complete svg: %q", svg)
	}
	// One track circle + one per slice.
	if n := strings.Count(svg, "<circle"); n != 1+len(rows) {
		t.Errorf("circle count = %d, want %d", n, 1+len(rows))
	}
	for _, c := range groupPalette[:len(rows)] {
		if !strings.Contains(svg, c) {
			t.Errorf("svg missing slice color %s", c)
		}
	}
}

func TestAllocationBarsHTML(t *testing.T) {
	rows := []models.GroupSummaryRow{
		row(t, "미국 & 성장", "42", "40", "1"), // over → 매도, name needs escaping
		row(t, "채권", "18", "20", "-1"),     // under → 매수
	}
	html := allocationBarsHTML(rows)
	if !strings.Contains(html, "미국 &amp; 성장") {
		t.Errorf("group name not HTML-escaped: %s", html)
	}
	if !strings.Contains(html, "매도") || !strings.Contains(html, "매수") {
		t.Errorf("missing action badges: %s", html)
	}
	if !strings.Contains(html, "width:42.0%") {
		t.Errorf("missing actual fill width: %s", html)
	}
	if !strings.Contains(html, "목표 40.0%") {
		t.Errorf("missing target label: %s", html)
	}
}

func TestDiffBadgeHTML(t *testing.T) {
	cases := []struct{ val, want string }{
		{"5", "매도"},
		{"-5", "매수"},
		{"0", "유지"},
	}
	for _, c := range cases {
		if got := diffBadgeHTML(mustDec(t, c.val)); !strings.Contains(got, c.want) {
			t.Errorf("diffBadgeHTML(%s) = %q, want contains %q", c.val, got, c.want)
		}
	}
}

func TestPnlAndFormatSignedKRW(t *testing.T) {
	s := &models.PortfolioSummary{
		TotalAssets:   numeric.FromInt(128400000),
		TotalInvested: numeric.FromInt(114200000),
	}
	pnl := pnlValue(s)
	if !pnl.Equal(numeric.FromInt(14200000).Decimal) {
		t.Errorf("pnlValue = %s, want 14200000", pnl.String())
	}
	if got := formatSignedKRW(pnl); got != "+₩14,200,000" {
		t.Errorf("formatSignedKRW(+) = %q", got)
	}
	if got := formatSignedKRW(numeric.FromInt(-5000)); got != "-₩5,000" {
		t.Errorf("formatSignedKRW(-) = %q", got)
	}
}

func TestTrendArrow(t *testing.T) {
	up := numeric.FromInt(1)
	down := numeric.FromInt(-1)
	flat := numeric.Zero
	if trendArrow(&up) != "▲" {
		t.Error("up arrow wrong")
	}
	if trendArrow(&down) != "▼" {
		t.Error("down arrow wrong")
	}
	if trendArrow(&flat) != "" || trendArrow(nil) != "" {
		t.Error("flat/nil should yield empty arrow")
	}
}

func TestPctWidthClamp(t *testing.T) {
	if got := pctWidth(mustDec(t, "150")); got != "100.0%" {
		t.Errorf("over-100 clamp: %q", got)
	}
	if got := pctWidth(mustDec(t, "-5")); got != "0.0%" {
		t.Errorf("negative clamp: %q", got)
	}
	if got := pctWidth(mustDec(t, "42.34")); got != "42.3%" {
		t.Errorf("rounding: %q", got)
	}
}
