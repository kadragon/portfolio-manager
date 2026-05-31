package kis

// IsDomesticTicker reports whether ticker is a domestic (KOSPI/KOSDAQ) code.
// Domestic codes are exactly 6 characters (e.g. "005930"). Overseas tickers
// (e.g. "AAPL") are typically 1–5 alphabetic characters.
func IsDomesticTicker(ticker string) bool {
	return len(ticker) == 6
}
