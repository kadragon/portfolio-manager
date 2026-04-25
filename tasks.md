# Tasks — Deferred from PR reviews

## From PR #69 (feat/per-account-rebalance) — 2026-04-24

- ~~**Optimize `_pick_next_group` sort key**~~ — Skipped. Sort key (`need_by_group[g]`) mutates every iteration; a priority queue would need decrease-key + stale-entry handling for a 5-element collection. O(5 log 5) per iteration is effectively free. (`rebalance_service.py:866-868`)

## Follow-up from feat/stock-service-extraction (PR #70)

- [ ] **Consolidate duplicate name-resolution flows** — `services/portfolio_service.py:144-169` and `services/kis_account_sync_service.py:138-145` duplicate `format_stock_name + update_name`. Migrate both to `StockService.resolve_and_persist_name`. Note: `kis_account_sync_service.py:138` is a stock-creation path (not update); needs a different interface or overloaded call.

## Review Backlog

### PR #70 — [REFACTOR] Extract StockService (2026-04-25)

- [ ] [debt] Redundant `StockService` initialization: created in `__init__` (no price_service) then recreated in `_setup_kis_client` (with price_service). Consider lazy init or a setter. (source: Codex, Gemini) — `container.py:99,286`
- [ ] [debt] `_build_stock_name_map` no-price_service path now iterates N times returning "" instead of early-returning 0 iterations. Semantically equivalent but drops the fast-exit. Consider a `has_price_service` property on StockService. (source: Claude) — `web/routes/accounts.py:20-32`
