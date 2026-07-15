---
name: portfolio-sync
description: >-
  Sync an account's holdings/cash from its linked broker (KIS or Toss),
  refresh stock prices, or backfill stock asset-class classification — via
  `go run ./cmd/pm`, the CLI that replaced the web UI and its removed
  background price-sync job (everything here is on-demand). Use for: "ISA
  동기화해줘", "TOSS 계좌 동기화", "계좌 동기화해줘", "가격 갱신해줘", "가격 동기화", "시세
  업데이트", "가격이 오래됐어", "자산분류 채워줘", "ETF인지 주식인지 분류해줘", or when another
  skill reports stale prices needing refresh. For sync/price/classification
  failures (auth errors, empty snapshots, KIS not configured, wrong
  environment), run the command here, then hand off to kis-debug if the fix
  isn't obvious.
---

# Portfolio Sync

Wraps `cmd/pm`'s broker-sync, price-sync, and stock-classification subcommands — the on-demand
replacements for the web app's sync buttons and its removed 15-minute background price ticker
and daily band-alert webhook. Nothing runs automatically anymore; every sync in this skill is
triggered by the user asking for it. Run all commands from the repo root.

## Sync one account's holdings/cash from its broker

```bash
go run ./cmd/pm sync -account "<name or unique substring>"
```

- Account resolution is exact-name-or-unique-substring, same as portfolio-data's `-account` —
  if it errors "ambiguous", list the matches and ask which one.
- Routing (KIS vs Toss) is automatic based on which link the account has (`KisAccountNo` vs
  `TossAccountSeq`) — you don't choose it.
- On success, the JSON result (`KisAccountSyncResult`) reports the KRW-converted aggregate
  `CashBalance`, raw `CashBalanceKRW` / `CashBalanceUSD`, optional `USDKRWRate`,
  `OldCashBalance`, `HoldingCount`, `CreatedStockCount`, and a `HoldingChanges` list — summarize
  the deltas for the user (what changed, not just "synced"). A `null` currency component means
  a legacy balance that has not yet been split by a broker sync; zero means a confirmed zero.
- **Empty-snapshot guard**: if the broker returns zero holdings while the account has existing
  ones on file, the sync stops with an error asking for confirmation, to avoid silently wiping
  real positions on a transient API hiccup. Ask the user: "정말 전량 매도해서 비어 있는 게 맞나요?"
  Only if they confirm, rerun with `-confirm-empty`:
  ```bash
  go run ./cmd/pm sync -account "<name>" -confirm-empty
  ```
  Never add `-confirm-empty` preemptively "just in case" — it removes a safety check that
  exists specifically to protect against data loss.
- Config errors ("KIS sync service not configured" / "Toss sync service not configured" /
  "has no KIS account number or Toss accountSeq linked") mean the account isn't linked yet —
  that's a portfolio-data `account update` task (`-kis-account-no`/`-toss-account-seq`), not a
  sync problem.

## Refresh prices

```bash
go run ./cmd/pm price-sync
```

Runs one full price-sync pass across all stocks (current prices + any missing historical
closes) and returns `{"status": "synced"}` on success — per-ticker fetch failures are logged
internally, not surfaced as a command failure, so a "synced" result doesn't guarantee every
single ticker actually updated. If a specific ticker still looks stale after this, that's a
kis-debug question (price client / exchange routing), not something to retry blindly.

There is no longer a background job doing this every 15 minutes — if the user or another skill
(e.g. rebalance-plan) needs current prices, run this explicitly first.

## Backfill a date range for one ticker

```bash
go run ./cmd/pm price-backfill -ticker <TICKER> -from YYYY-MM-DD -to YYYY-MM-DD
```

Fetches every daily close KIS has for that ticker in `[from, to]` and saves whatever isn't
already cached in `stock_prices` — use this when a ticker's history has a gap (stale since a
past date, e.g. an exchange-routing bug that silently stopped updating it) or when a "how did
this move over period X" question needs days `price-sync`'s fixed 1y/6m/1m/1d checkpoints don't
cover. Returns `{"Ticker", "Requested", "Saved", "Skipped"}` — `Skipped` counts dates already
cached (existing rows are never overwritten; past data is immutable). Internally chunks ranges
over ~90 days into multiple KIS calls (its period-endpoint caps around 100 rows per call), with
the same `syncCallDelay` pacing as `price-sync`.

Only takes one ticker at a time — for many tickers, loop the command per ticker (same
KIS-rate-limit caution as everywhere else in this file: don't fire them back-to-back with no
delay). If `Saved` comes back 0 with `Requested` also 0, KIS returned nothing for that ticker/
range — hand off to kis-debug rather than assuming the range was simply already complete.

## Ad-hoc historical price lookups ("이번 주 얼마나 올랐어?")

`stock_prices` (`internal/db/schema.sql`) keeps one row per `(ticker, price_date)` and
accumulates real daily closes over time — it is not overwritten across dates. For a quick read
of what's already cached (no KIS call), read `.data/portfolio.db` directly:

```bash
sqlite3 -separator '|' .data/portfolio.db \
  "SELECT ticker, price_date, price FROM stock_prices WHERE price_date >= date('now','-10 day') ORDER BY ticker, price_date;"
```

If a ticker is missing the dates you need, run `price-backfill` for it first (above), then
re-query. Weight per-ticker % changes by current `ValueKRW` (from `dashboard`) for a
portfolio-level estimate; call out that it's an approximation if quantities may have changed
over the window or USD/KRW moved.

## Backfill asset-class classification

```bash
go run ./cmd/pm classify-stocks
```

Classifies unclassified stocks as `"etf"` or `"stock"` via KIS lookups (needed so IRP/연금
account eligibility checks — ETF/fund-only — work correctly). Returns a
`StockClassificationResult` with `Total`/`Classified`/`Skipped`/`Failed` counts — report the
`Failed` count to the user if non-zero (those stocks got a non-fatal sentinel tag, not a real
classification, and may need a manual look).

## KIS rate limit

KIS token issuance is capped at once per minute (see AGENTS.md). Don't loop `sync`/`price-sync`/
`classify-stocks` calls back-to-back hoping a transient failure clears — if one fails, surface
the error and wait for the user's next instruction rather than immediately retrying.

## Failure modes to avoid

- Don't run `sync`, get an error, and silently fall back to `portfolio-data` manual edits to
  "fix" what looks wrong — the sync error is real signal; report it as-is.
- Don't chain `price-sync` → `classify-stocks` → `sync` speculatively when the user only asked
  for one of them — each is a separate, deliberate action.
- If the error text doesn't map cleanly to one of the cases above (unexpected KIS error code,
  HTTP failure, garbled response), hand off to the kis-debug skill rather than guessing at a fix.
