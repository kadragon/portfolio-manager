# Backlog

## Review Backlog

- [ ] [REFACTOR] Batch benchmark deposit-date history via `GetHistoricalRange` — `internal/services/price_sync_service.go` issues one `GetHistoricalClose` per deposit date per timing-matched ticker (plus up to ~10 business-day walk-back calls when a date is empty), at `syncCallDelay` = 200ms each. A long deposit history makes `pm price-sync` take minutes. `BackfillRange` already chunks a date range into ≤90-day `GetHistoricalRange` calls; deposit dates clustered within one window could share a single request. Deferred from PR #174 codex review (P2) — needs a windowing rule that does not fetch years of daily closes for a handful of dates.
