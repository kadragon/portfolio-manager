---
name: verify-kis-pricing
description: Verify KIS pricing, exchange normalization, and order routing behavior after KIS/price changes.
---

# Verify KIS Pricing

## Purpose
1. Ensure price parsing matches KIS payload variations for domestic/overseas.
2. Validate exchange normalization (NAS/NYS/AMS ↔ NASD/NYSE/AMEX) in PriceService and order routing.
3. Confirm token refresh retry behavior remains intact for KIS clients.
4. Verify price cache writes store canonical exchanges.

## When to Run
- Changes in `src/portfolio_manager/services/kis/` or `src/portfolio_manager/services/price_service.py`.
- KIS API response parsing or exchange code handling changes.
- Order routing changes involving overseas exchanges.
- Token refresh or retry logic updates.

## Related Files

| File | Purpose |
| --- | --- |
| `src/portfolio_manager/services/price_service.py` | Exchange normalization and price cache behavior |
| `src/portfolio_manager/services/kis/kis_price_parser.py` | KIS price parsing helpers |
| `src/portfolio_manager/services/kis/kis_domestic_price_client.py` | Domestic price fetch + parser usage |
| `src/portfolio_manager/services/kis/kis_overseas_price_client.py` | Overseas price fetch + parser usage |
| `src/portfolio_manager/services/kis/kis_unified_price_client.py` | Exchange prioritization and routing |
| `src/portfolio_manager/services/kis/kis_unified_order_client.py` | Order exchange normalization |
| `src/portfolio_manager/services/kis/kis_base_client.py` | Shared retry logic |
| `tests/services/test_price_service.py` | PriceService behavior checks |
| `tests/services/test_price_service_memory_cache.py` | In-memory cache exchange behavior |
| `tests/services/kis/test_kis_price.py` | Price client and parser tests |
| `tests/services/kis/test_kis_order_client.py` | Order exchange mapping tests |
| `tests/test_kis_token_refresh.py` | Token retry behavior |

## Workflow

### Step 1: Confirm exchange normalization in PriceService
**Tool:** Read

- Verify `PriceService` maps `NAS/NYS/AMS` to `NASD/NYSE/AMEX` for returned exchange and cache writes.
- Confirm preferred exchange inputs are mapped to price client codes via `_to_price_exchange`.

**PASS:** `_ORDER_EXCHANGE_MAP` and `_PRICE_EXCHANGE_MAP` exist and are applied in `get_stock_price`.
**FAIL:** Any direct use of `NAS/NYS/AMS` in cache writes or return values without normalization.

### Step 2: Verify parser coverage for payload variants
**Tool:** Read

- Inspect `parse_korea_price` and `parse_us_price` for list output handling and missing symbol/name fallback.

**PASS:** Parser handles list outputs and optional `symbol`/`exchange` inputs.
**FAIL:** Parser assumes required keys without fallback.

### Step 3: Validate KIS price clients use the parser helpers
**Tool:** Read

- Confirm domestic and overseas price clients delegate to `parse_korea_price` / `parse_us_price`.

**PASS:** No duplicated parsing logic inside `fetch_current_price` and retry variants.
**FAIL:** Inline parsing still present.

### Step 4: Validate order exchange mapping for overseas orders
**Tool:** Read

- Ensure `KisUnifiedOrderClient` normalizes exchange to order code via a mapping table.

**PASS:** Mapping includes NAS→NASD, NYS→NYSE, AMS→AMEX.
**FAIL:** `exchange or "NASD"` without normalization.

### Step 5: Run targeted tests
**Tool:** Bash

```bash
uv run pytest -q \
  --no-cov \
  tests/services/test_price_service.py \
  tests/services/test_price_service_memory_cache.py \
  tests/services/kis/test_kis_price.py \
  tests/services/kis/test_kis_order_client.py \
  tests/test_kis_token_refresh.py
```

**PASS:** All tests green.
**FAIL:** Any failures -> inspect exchange normalization and parser usage.

## Output Format

| Check | Status | Notes |
| --- | --- | --- |
| Exchange normalization | PASS/FAIL |  |
| Parser coverage | PASS/FAIL |  |
| Clients use parser | PASS/FAIL |  |
| Order mapping | PASS/FAIL |  |
| Tests | PASS/FAIL |  |

## Exceptions

1. Demo-only or integration tests that require KIS credentials can be skipped.
2. Temporary fallback to empty names is allowed when KIS returns blank fields.
3. In-memory cache may return stale exchange values within a single session; this does not imply DB cache regression.
