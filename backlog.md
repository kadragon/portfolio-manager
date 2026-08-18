# Backlog

## Populate benchmark price history at deposit dates

`pm price-sync` stores only the 1y/6m/1m/1d checkpoints plus today's close, so with the new
31-day staleness guard (`maxBenchmarkPriceGap`, `internal/services/portfolio_service.go`) a
timing-matched benchmark reports `null` whenever a deposit falls in a hole between checkpoints —
the common case on a DB that has only ever run `price-sync`. `benchmarkHistoricalDates` should
also fetch each deposit date for benchmark tickers, or `price-sync` should auto-backfill the
deposit range for them. Deferred from PR #173 review: it multiplies KIS calls by the deposit
count and needs a rate-limit/pagination design, not a one-line change.

Source: task-review of PR #173 (Codex P2 + contest round, confirmed against `.data/portfolio.db`).
