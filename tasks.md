# Tasks — Deferred from PR reviews

## From PR #69 (feat/per-account-rebalance) — 2026-04-24

- ~~**Optimize `_pick_next_group` sort key**~~ — Skipped. Sort key (`need_by_group[g]`) mutates every iteration; a priority queue would need decrease-key + stale-entry handling for a 5-element collection. O(5 log 5) per iteration is effectively free. (`rebalance_service.py:866-868`)

## Follow-up from feat/stock-service-extraction (PR #70)

- [ ] **Consolidate duplicate name-resolution flows** — `services/portfolio_service.py:144-169` and `services/kis_account_sync_service.py:138-145` duplicate `format_stock_name + update_name`. Migrate both to `StockService.resolve_and_persist_name`.
