---
name: portfolio-sync
description: >-
  Sync an account's holdings/cash from its linked broker (KIS or Toss),
  refresh stock prices, or backfill stock asset-class classification — via
  `go run ./cmd/pm`, the CLI that replaced the app's web UI (and its removed
  15-minute background price-sync job; everything here is now on-demand
  only). Use whenever the user asks to sync/refresh broker or price data:
  "ISA 동기화해줘", "TOSS 계좌 동기화", "계좌 동기화해줘", "가격 갱신해줘", "가격 동기화",
  "시세 업데이트", "가격이 오래됐어", "자산분류 채워줘", "ETF인지 주식인지 분류해줘", or when
  another skill (e.g. rebalance-plan) reports stale prices and needs them
  refreshed before it can proceed. For sync/price/classification *failures*
  (auth errors, empty snapshots, KIS not configured, wrong environment), use
  this skill to run the command and read the error, then hand off to
  kis-debug for root-cause diagnosis if the fix isn't obvious from the error
  text alone.
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
- On success, the JSON result (`KisAccountSyncResult`) reports `CashBalance`, `OldCashBalance`,
  `HoldingCount`, `CreatedStockCount`, and a `HoldingChanges` list — summarize the deltas for
  the user (what changed, not just "synced").
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
