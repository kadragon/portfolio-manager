package templates

import (
	"html"
	"strconv"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/accountformat"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
)

// groupPalette is a tonal, on-brand palette for allocation slices/bars. It stays
// inside the Coinbase-light voice (blue family + ink + gold + slate) and
// deliberately avoids success-green / error-red so allocation color never reads
// as a gain/loss signal.
var groupPalette = []string{
	"#0052ff", // primary blue
	"#0a0b0d", // ink
	"#f4b000", // gold accent
	"#5b616e", // slate
	"#7aa2ff", // light blue
	"#b08400", // deep gold
	"#9aa0ab", // light slate
	"#003ecc", // deep blue
}

// groupColor returns a stable slice color for the i-th group (cycles).
func groupColor(i int) string {
	return groupPalette[i%len(groupPalette)]
}

// pctFloat clamps a decimal percentage to [0,100] and returns it as a float.
func pctFloat(d numeric.Decimal) float64 {
	f := d.InexactFloat64()
	if f < 0 {
		return 0
	}
	if f > 100 {
		return 100
	}
	return f
}

// pctWidth returns a clamped CSS width string like "42.1%" for a percentage value.
func pctWidth(d numeric.Decimal) string {
	return strconv.FormatFloat(pctFloat(d), 'f', 1, 64) + "%"
}

// pctLeft is an alias kept readable at call sites for the target marker position.
func pctLeft(d numeric.Decimal) string { return pctWidth(d) }

// donutSVG renders an allocation donut as a self-contained inline SVG string.
// Each slice length is the group's actual share of stock value (ActualPct, which
// sums to ~100 across rows). Uses the r=15.915 circumference≈100 trick so dash
// lengths map directly to percentages. Returns "" when there is nothing to draw.
func donutSVG(rows []models.GroupSummaryRow) string {
	if len(rows) == 0 {
		return ""
	}
	var total float64
	for _, r := range rows {
		total += pctFloat(r.ActualPct)
	}
	if total <= 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(`<svg viewBox="0 0 42 42" class="w-full h-full" role="img" aria-label="그룹별 자산 배분 도넛 차트">`)
	// Track ring.
	b.WriteString(`<circle cx="21" cy="21" r="15.915" fill="none" stroke="#eef1f5" stroke-width="5"></circle>`)
	b.WriteString(`<g transform="rotate(-90 21 21)">`)

	cumulative := 0.0
	for i, r := range rows {
		// Normalize so slices fill the ring even if pcts don't sum to exactly 100.
		p := pctFloat(r.ActualPct) / total * 100
		if p <= 0 {
			continue
		}
		dash := strconv.FormatFloat(p, 'f', 3, 64)
		gap := strconv.FormatFloat(100-p, 'f', 3, 64)
		offset := strconv.FormatFloat(-cumulative, 'f', 3, 64)
		b.WriteString(`<circle cx="21" cy="21" r="15.915" fill="none" stroke="`)
		b.WriteString(groupColor(i))
		b.WriteString(`" stroke-width="5" stroke-dasharray="`)
		b.WriteString(dash)
		b.WriteByte(' ')
		b.WriteString(gap)
		b.WriteString(`" stroke-dashoffset="`)
		b.WriteString(offset)
		b.WriteString(`"></circle>`)
		cumulative += p
	}
	b.WriteString(`</g></svg>`)
	return b.String()
}

// allocationBarsHTML renders the per-group target-vs-actual bar list as a raw
// HTML string. Built in Go (rather than templ markup) so inline width/color
// styles bypass templ's CSS attribute sanitization, mirroring donutSVG. Each row
// shows actual vs target share, an over/under-target marker, and a 매도/매수/유지
// action badge consistent with the rebalance allocation table.
func allocationBarsHTML(rows []models.GroupSummaryRow) string {
	var b strings.Builder
	b.WriteString(`<ul class="space-y-3.5 w-full">`)
	for i, r := range rows {
		color := groupColor(i)
		name := html.EscapeString(r.Group.Name)
		b.WriteString(`<li>`)
		b.WriteString(`<div class="flex items-center justify-between gap-3 text-sm mb-1.5">`)
		b.WriteString(`<span class="flex items-center gap-2 font-medium text-base-content min-w-0">`)
		b.WriteString(`<span class="inline-block w-2.5 h-2.5 rounded-full shrink-0" style="background-color:` + color + `"></span>`)
		b.WriteString(`<span class="truncate">` + name + `</span>`)
		b.WriteString(`</span>`)
		b.WriteString(`<span class="flex items-center gap-2 font-mono text-xs whitespace-nowrap">`)
		b.WriteString(`<span class="text-base-content">` + html.EscapeString(accountformat.FormatPercent(r.ActualPct)) + `</span>`)
		b.WriteString(`<span class="text-base-content/40">목표 ` + html.EscapeString(accountformat.FormatPercent(r.TargetPct)) + `</span>`)
		b.WriteString(diffBadgeHTML(r.DiffVal))
		b.WriteString(`</span></div>`)
		// Track + actual fill + target marker.
		b.WriteString(`<div class="relative h-2 rounded-full bg-base-200">`)
		b.WriteString(`<div class="absolute inset-y-0 left-0 rounded-full" style="width:` + pctWidth(r.ActualPct) + `;background-color:` + color + `"></div>`)
		b.WriteString(`<div class="absolute -inset-y-0.5 w-px bg-base-content/60" style="left:` + pctLeft(r.TargetPct) + `" title="목표 ` + html.EscapeString(accountformat.FormatPercent(r.TargetPct)) + `"></div>`)
		b.WriteString(`</div></li>`)
	}
	b.WriteString(`</ul>`)
	return b.String()
}

// diffBadgeHTML returns the rebalance action badge for a group value gap:
// positive (over target) → 매도, negative (under target) → 매수, zero → 유지.
func diffBadgeHTML(diffVal numeric.Decimal) string {
	switch {
	case diffVal.IsPositive():
		return `<span class="badge badge-error badge-sm">매도</span>`
	case diffVal.IsNegative():
		return `<span class="badge badge-success badge-sm">매수</span>`
	default:
		return `<span class="badge badge-ghost badge-sm">유지</span>`
	}
}

// pnlValue is the absolute profit/loss: total assets minus invested principal.
func pnlValue(s *models.PortfolioSummary) numeric.Decimal {
	return numeric.Wrap(s.TotalAssets.Sub(s.TotalInvested.Decimal))
}

// formatSignedKRW formats a KRW amount with an explicit leading + for gains.
// FormatKRW already prefixes - for losses.
func formatSignedKRW(d numeric.Decimal) string {
	if d.IsPositive() {
		return "+" + accountformat.FormatKRW(d)
	}
	return accountformat.FormatKRW(d)
}

// trendArrow returns a directional glyph for a signed rate, or "" when nil/flat.
func trendArrow(rate *numeric.Decimal) string {
	if rate == nil {
		return ""
	}
	if rate.IsPositive() {
		return "▲"
	}
	if rate.IsNegative() {
		return "▼"
	}
	return ""
}
