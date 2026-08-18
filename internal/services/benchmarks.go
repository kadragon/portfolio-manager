package services

// BenchmarkMode selects how dashboard benchmark returns are computed.
type BenchmarkMode string

const (
	// BenchmarkModeLumpSum values a single hypothetical purchase made on the
	// first deposit date. Its returns are price returns in each benchmark's own
	// currency, so a USD-listed benchmark carries no FX effect.
	BenchmarkModeLumpSum BenchmarkMode = "lump-sum"
	// BenchmarkModeTimingMatched replays every deposit into the benchmark on the
	// deposit's own date, producing a money-weighted return directly comparable
	// with PortfolioSummary.ReturnRate.
	BenchmarkModeTimingMatched BenchmarkMode = "timing-matched"
)

type benchmarkSpec struct {
	label             string
	ticker            string
	preferredExchange string
}

var dashboardBenchmarks = []benchmarkSpec{
	{label: "S&P 500", ticker: "SPY", preferredExchange: "AMEX"},
	{label: "Nasdaq", ticker: "QQQ", preferredExchange: "NASD"},
	{label: "KOSPI", ticker: "226490"},
}

// timingMatchedBenchmarks mirrors dashboardBenchmarks with KRW-listed proxies.
// The timing-matched return is compared against a KRW portfolio return, so a
// USD-listed proxy would drop the FX component of the comparison; these three
// ETFs quote in KRW and therefore embed it.
var timingMatchedBenchmarks = []benchmarkSpec{
	{label: "S&P 500 (KRW)", ticker: "360750"},
	{label: "Nasdaq (KRW)", ticker: "368590"},
	{label: "KOSPI", ticker: "226490"},
}

// allBenchmarks is every benchmark ticker the dashboard can ask for, in either
// mode. Price sync must cover this union: a timing-matched proxy that is never
// fetched has neither a current price nor historical closes, so its return is
// reported as nil on an otherwise healthy DB.
func allBenchmarks() []benchmarkSpec {
	specs := make([]benchmarkSpec, 0, len(dashboardBenchmarks)+len(timingMatchedBenchmarks))
	seen := make(map[string]bool, cap(specs))
	for _, set := range [][]benchmarkSpec{dashboardBenchmarks, timingMatchedBenchmarks} {
		for _, b := range set {
			if seen[b.ticker] {
				continue
			}
			seen[b.ticker] = true
			specs = append(specs, b)
		}
	}
	return specs
}
