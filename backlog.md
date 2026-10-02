# Backlog

## Toss API Coverage

Gap vs the official spec (`https://openapi.tossinvest.com/openapi-docs/latest/openapi.json`, parsed 2026-10-02): 30 of 36 operations implemented in `internal/toss/`; the 6 below — all tag Stock Info — are missing.

- [ ] [FEAT] Toss `GET /api/v1/stocks/all` (`listStocks`) → `pm toss stocks-all -market KOSPI|KOSDAQ|NYSE|NASDAQ|AMEX|KR_ETC|US_ETC [-status SCHEDULED|ACTIVE|DELISTED] [-security-type ...] [-common-share]` — market-wide tradable-symbol universe (`market` required). Use case: verify a ticker is Toss-tradable/ACTIVE before `stock add` or a buy, without per-symbol `GetStocks` lookups.
- [ ] [FEAT] Toss KR per-stock flow series — five endpoints sharing one shape (`symbol` required, optional `count`, `until`; daily time series; KR symbols only): `/api/v1/stocks/{symbol}/investor-trading` (`getStockInvestorTrading`), `/program-trades` (`getStockProgramTrades`), `/short-selling` (`getStockShortSelling`), `/credit-trades` (`getStockCreditTrades`), `/securities-lending` (`getStockSecuritiesLending`). Mirror `GetMarketIndicatorInvestorTrading` in `internal/toss/market_indicators.go` (path-escaped symbol, count/until query) and expose as `pm toss stock-<kind> -symbol T [-count N] [-until YYYY-MM-DD]`. Read-only market data; low priority for an ETF portfolio — implement together as one slice.
