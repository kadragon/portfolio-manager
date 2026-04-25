<!--
Schema / lifecycle:
  Active sprint  — open [ ] items grouped by PR or feature branch below this header.
  Dormant        — no active sprint; file starts with "# (dormant)" and retains only open [debt/blocked] items.
-->

# Tasks — Deferred from PR reviews

## From PR #69 (feat/per-account-rebalance) — 2026-04-24

- ~~**Optimize `_pick_next_group` sort key**~~ — Skipped. Sort key (`need_by_group[g]`) mutates every iteration; a priority queue would need decrease-key + stale-entry handling for a 5-element collection. O(5 log 5) per iteration is effectively free. (`rebalance_service.py:866-868`)

## Follow-up from feat/stock-service-extraction (PR #70)

- [x] **Consolidate duplicate name-resolution flows** — Added `StockService.persist_name(stock, raw_name)` shared helper; `portfolio_service.py` and `kis_account_sync_service.py` delegate to it when `stock_service` is injected. Creation path in `kis_account_sync_service.py:138` kept using `format_stock_name` directly (no Stock instance yet). (#71)

## Review Backlog

### PR #70 — [REFACTOR] Extract StockService (2026-04-25)

- [x] [debt] Redundant `StockService` initialization: replaced second construction with `set_price_service()` setter — single object identity across container lifecycle. (#71)
- [x] [debt] `_build_stock_name_map` no-price_service path: added `has_price_service` property to `StockService`. Fast-exit in routes left unchanged — existing test confirmed that stocks with already-persisted names should be returned even when `price_service` is unavailable. (#71)

## Review Backlog

### PR #71 — [REFACTOR] StockService API (2026-04-25)

- [ ] [debt/blocked] Remove `else` fallback in `portfolio_service.py:149-152` — multiple tests instantiate `PortfolioService` without `stock_service` (`test_portfolio_service.py:62` etc.); removal requires refactoring those tests first. (source: Claude)
- [ ] [doc/blocked] `persist_name` docstring: update dual-path description when the else-branch above is removed. (source: Claude)
