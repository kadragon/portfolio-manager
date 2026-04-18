# Tasks — Deferred from PR reviews

## From PR #67 (feat/stock-name-wave2) — 2026-04-18

- [ ] **GP-1 spirit: move `stock_repository.update_name` out of `web/routes/accounts.py`** — Route currently calls repository directly. Arch test passes, but principle says web → service → repository. Introduce `StockService.ensure_name(stock, resolved_name)` or similar. (Claude review, important)
- [ ] **Add ETF-suffix edge case test for `_build_stock_name_map`** — Verify `format_stock_name` truncation is what gets persisted, e.g. `"KODEX 200증권상장지수투자신탁(주식)"`. (Claude review, nice-to-have)
- [ ] **Add test covering non-`ValueError` exception path in `_build_stock_name_map`** — Verify page renders + warning logged when `get_stock_price` raises `RuntimeError`/`KeyError`/etc. (Claude review, nice-to-have)
- [ ] **Fix `FakeStockRepository.update` to preserve `name`** — `tests/web/conftest.py:121–134` drops `name` on ticker update. Latent test-double bug, not caused by PR #67. (Claude review, nice-to-have)
