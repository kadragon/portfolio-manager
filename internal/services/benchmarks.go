package services

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
