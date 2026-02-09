# AGENTS.md

## Development Notes
- Reference: https://github.com/koreainvestment/open-trading-api (KIS Open Trading API examples/specs) for ongoing implementation alignment.
- KIS token issuance is rate-limited (1 per minute); token cache stored at `.data/kis_token.json`.
- Local verification script: `scripts/check_kis_domestic_price.py` (loads `.env`, fetches token if needed, calls domestic price API).
- Local verification script: `scripts/check_kis_overseas_price.py` (loads `.env`, fetches token if needed, calls overseas price API; use `.venv/bin/python`).
- `KIS_ENV` accepts `real/prod` and `demo/vps/paper` (also tolerates `real/prod` form and whitespace).
- Overseas current price endpoint: `/uapi/overseas-price/v1/quotations/price` with TR ID `HHDFS00000300`.

## Supabase Integration
- Supabase credentials stored in `.env`: `SUPABASE_URL` and `SUPABASE_KEY`.
- Auto-resume for paused projects: Set `SUPABASE_ACCESS_TOKEN` (Personal Access Token from https://supabase.com/dashboard/account/tokens) to automatically restore paused Supabase projects on connection failure.
- Database schema:
  - `groups` table: stores stock groups (id, name, target_percentage, created_at, updated_at)
  - `stocks` table: stores stock tickers (id, ticker, group_id, created_at, updated_at)
  - Relationship: groups 1:N stocks
- `accounts` table: stores brokerage accounts (id, name, cash_balance, created_at, updated_at)
- `holdings` table: stores account holdings (id, account_id, stock_id, quantity, created_at, updated_at)
- Relationship: accounts 1:N holdings, stocks 1:N holdings
- Migration files:
  - `supabase/migrations/20260103000000_create_groups_and_stocks.sql`
  - `supabase/migrations/20260103010000_create_accounts_and_holdings.sql`
  - `supabase/migrations/20260104000000_add_target_percentage_to_groups.sql`
  - `supabase/migrations/20260104010000_create_deposits.sql`
  - `supabase/migrations/20260104020000_alter_deposits_global.sql`
- Repositories implemented:
  - `GroupRepository`: create(), list_all(), update(), delete()
  - `StockRepository`: create(), list_by_group()
  - `AccountRepository`: create(), list_all(), delete_with_holdings()
  - `HoldingRepository`: create(), list_by_account(), delete_by_account(), get_aggregated_holdings_by_stock()
  - `DepositRepository`: create(), update(), list_all(), get_by_date(), delete(), get_total(), get_first_deposit_date()
- Services implemented:
  - `PortfolioService`: get_holdings_by_group() - aggregates holdings across accounts by stock and groups them by group
  - `PortfolioService`: get_portfolio_summary() - aggregates holdings with real-time price, valuation, return rates (including annualized)
  - `PriceService`: get_stock_price(ticker) - fetches current stock price from price client
  - `KisUnifiedPriceClient`: get_price(ticker) - routes to domestic/overseas KIS API based on ticker format
- Supabase client factory: `src/portfolio_manager/services/supabase_client.py`
- Data models: `src/portfolio_manager/models/group.py` (target_percentage 포함), `src/portfolio_manager/models/stock.py`, `src/portfolio_manager/models/account.py`, `src/portfolio_manager/models/holding.py`
- Test coverage: `tests/test_group_repository.py`, `tests/test_stock_repository.py`, `tests/test_account_repository.py`, `tests/test_account_delete_cascade.py`, `tests/test_holding_repository.py`, `tests/test_holding_quantity_decimal.py`, `tests/test_holding_aggregation.py`, `tests/test_portfolio_service.py`, `tests/test_rich_dashboard.py`

## Strategic Insights
- Rich-only CLI replaces Textual screens; menu navigation and prompts drive group/stock flows.
- Group selection now leads to a stock menu loop with table-based rendering and back navigation.
- List selections for groups/stocks/accounts/holdings use prompt_toolkit choice (arrow-key) instead of numeric input.
- Main menu displays unified portfolio dashboard in single table with price and valuation information.
- Dashboard shows: Group, Ticker, Quantity, Price, Value (USD는 KRW 환산), and Total Portfolio Value.
- 대시보드에 그룹별 합계/비중/목표 대비 리밸런스 요약 표를 추가하고, Buy/Sell은 아이콘+색상으로 표시하며 금액 컬럼을 분리했다.
- Currency symbols display based on stock market: ₩ for KRW (domestic), $ for USD (overseas).
- 해외 종목명 누락 시 티커를 Name 컬럼에 표시하고, KIS 응답의 다양한 이름 필드로 보완한다.
- 그룹 목록에 목표 비중(%)을 함께 표시하고 추가/수정 시 입력을 받는다.
- 입금 내역은 계좌와 무관하게 일자별로 1건만 허용하며, 중복 날짜는 수정 흐름으로 유도한다.
- 리밸런싱 메뉴에서 그룹 목표 비중과 현재 평가액 차이를 기반으로 개별 주식 매매 추천을 제공한다.
- 리밸런싱 추천 표에 주식명/수량을 함께 표시하고 Priority 컬럼은 제거했다.
- 대시보드 주식명은 길이 제한 없이 전체 표시한다.

## Governance Updates
- Authentication clients now share the `AuthClient` interface to decouple token management from a concrete provider.
- Rich CLI flows and account/holding repositories are test-backed to lock in prompt/flow behavior and data parsing.
- Choice-based selection helpers cover group/account/stock/holding lists to standardize CLI selection inputs.
- Portfolio aggregation logic is encapsulated in PortfolioService with comprehensive test coverage for cross-account holding summation.
- Real-time pricing integrated via KIS API with automatic market detection (6-character codes = domestic, including alphanumeric like "0052D0"; alphabetic = overseas).
- Dashboard gracefully degrades to quantity-only display if KIS credentials unavailable or price fetch fails.
- Market detection uses length-based logic: 6-character tickers route to domestic API, others to overseas API (supports both pure numeric and alphanumeric Korean stock codes).
- Currency information flows from PriceQuote through PriceService and StockHoldingWithPrice to dashboard rendering with appropriate symbols (₩/$ based on KRW/USD).
- 해외 가격 조회는 거래소(NAS/NYS/AMS)를 순차 조회하며, 비어있는 종목명은 대체 필드로 채운다.
- USD 보유분은 `value_krw`에 환산 평가액을 저장해 대시보드에서 KRW 기준으로 출력한다.
- USD/KRW 환율 조회는 EXIM에서 USD 누락 시 최근 7일 내 직전 영업일로 자동 재시도한다.
- KIS 클라이언트는 공통 `KisBaseClient`에서 헤더 구성 및 환경별 TR ID 매핑을 공유한다.
- 투자 원금 합계는 계좌별 합산이 아니라 전체 deposits 합계로 계산된다.
- KIS API 토큰 만료 시 자동 갱신: `is_token_expired_error()`로 500 에러 중 토큰 만료(msg_cd: 'EGW00123')를 감지하고, `KisDomesticPriceClient`와 `KisOverseasPriceClient`는 `token_manager`가 제공되면 자동으로 토큰 재발급 및 재시도를 수행한다. 이를 통해 토큰 만료로 인한 일시적 실패를 사용자 개입 없이 복구한다.
- 대시보드 투자 요약(Total Summary)을 Rich Panel로 개선하고, 연환산 수익률(Annualized Return Rate) 표시 기능을 추가했다. 최초 입금일로부터 경과 일수를 기준으로 ((총자산/투자원금)^(365/경과일수) - 1) × 100 공식으로 계산한다.
- `RebalanceService`가 그룹별 과대/과소 비중을 계산하고 개별 주식 매매 추천을 생성한다. 매도 시 해외주식(USD)을 우선 추천하고, 매수 시 국내주식(KRW)을 우선 추천하여 환전 비용과 세금을 최소화한다.
- 주식명 표기 시 "증권상장지수투자신탁(주식)" 접미어를 제거하는 규칙을 CLI와 PortfolioService에 적용했다.

## Governance Updates
- Added `is_domestic_ticker()` helper in KIS services to centralize market detection by ticker length and keep routing logic consistent across clients.
- PriceService now provides `get_stock_change_rates()` to compute 1Y/6M/1M change percentages from historical closes with date-shift helpers.
- Dashboard now renders 1Y/6M/1M change-rate columns per holding when change rate data is available.
- Added KIS historical close fetch methods for domestic and overseas price clients to support date-based close lookups.
- Change-rate calculation now adjusts target dates that fall on weekends to the previous business day before fetching historical closes.
- Added a lightweight CLI integration test to ensure the main loop renders the dashboard once and exits cleanly when quit is selected.
- KisUnifiedPriceClient now exposes `get_historical_close()` with domestic routing and overseas exchange fallback to support change-rate queries.
- KisUnifiedPriceClient now skips HTTPStatusError failures for overseas quotes and falls back to the next exchange.
- Added `stocks.exchange` migration to cache the preferred overseas exchange per ticker.
- Stock model and repository now track a preferred overseas exchange via `stocks.exchange` and support updating it.
- KisUnifiedPriceClient now accepts an optional preferred exchange to prioritize NAS/NYS/AMS ordering for overseas quotes.
- KisUnifiedPriceClient historical close lookup now accepts a preferred exchange for prioritized overseas history queries.
- PriceService now passes preferred exchanges into KIS clients and returns the resolved exchange so PortfolioService can persist exchange cache updates.
- Overseas historical close lookup now skips HTTPStatusError responses and falls back to the next exchange.
- Added `stock_prices` table and repositories to cache daily price snapshots per ticker/date.
- PriceService now reuses cached prices for the day and caches non-zero quotes from live fetches.
- Historical close lookups now use the same daily cache and skip cache writes on errors or zero prices.
- Supabase 자동 resume: `get_supabase_client()`가 연결 실패 시 `SUPABASE_ACCESS_TOKEN`이 설정되어 있으면 Management API로 paused 프로젝트를 자동 복구하고 재연결을 시도한다.

## 2026-01-29

### Decision/Learning
KIS domestic historical close fetches a 7-day range ending on the target date and selects the matching date or the most recent available close.

### Reason
The domestic daily API can omit prices on holidays; a range ensures a prior trading day is returned.

### Impact
Tests and callers should expect `FID_INPUT_DATE_1` to be `target_date - 7 days`.

## 2026-01-31 (In-memory Caching)

### Decision/Learning
Added in-memory caching to `PriceService` and `ExchangeRateService` for API responses within a single program execution.

### Reason
Dashboard navigation and screen transitions cause repeated API calls for the same data. In-memory cache eliminates redundant API and DB queries during a session.

### Impact
- `PriceService._price_cache`: Caches (price, currency, name, exchange) tuples by ticker
- `PriceService._change_rates_cache`: Caches 1Y/6M/1M change rate dicts by (ticker, as_of) tuple
- `ExchangeRateService._cached_rate`: Caches USD/KRW rate (converted from frozen dataclass to regular class)
- Cache flow: Memory cache -> DB cache -> API call

## 2026-01-31 (Integration Tests)

### Decision/Learning
Marked the KIS script test as `integration` and default pytest runs exclude integration; the test skips if KIS credentials are missing.

### Reason
External network tests are flaky and require secrets that are not always available.

### Impact
Run integration tests explicitly with `-m integration` once credentials are configured.

## 2026-02-05 (Performance Flags)

### Decision/Learning
Added an `include_change_rates` flag to `PortfolioService.get_portfolio_summary` to allow callers to skip change-rate lookups.

### Reason
Change-rate lookups can trigger multiple historical price requests per holding, which is expensive for dashboard renders.

### Impact
Callers can disable change-rate fetching when fast rendering is preferred.

## 2026-02-05 (Dashboard Performance)

### Decision/Learning
Main dashboard rendering in `cli.main` now skips change-rate lookups by default.

### Reason
Change-rate lookups require multiple historical price requests per holding and slow down the main loop.

### Impact
If change-rate data is needed, callers must explicitly enable it.

## 2026-02-05 (Stock Loading)

### Decision/Learning
PortfolioService now loads all stocks once and groups them in memory to avoid per-group queries.

### Reason
Fetching stocks per group caused N+1 Supabase calls on each dashboard render.

### Impact
Provide `StockRepository.list_all()` and use it for portfolio aggregation.

## 2026-02-05 (Holding Aggregation)

### Decision/Learning
Holding aggregation now relies on Supabase RPC `aggregate_holdings_by_stock`.

### Reason
Server-side aggregation avoids loading the full holdings table on each dashboard render.

### Impact
Migration adds `aggregate_holdings_by_stock` RPC; it must return `stock_id` and `quantity`.

## 2026-02-05 (Preferred Exchange Fallback)

### Decision/Learning
Overseas quote lookup now returns after the preferred exchange succeeds without falling back to other exchanges.

### Reason
Fallbacks on non-error responses multiply HTTP calls and slow down dashboard pricing.

### Impact
Preferred exchange is treated as authoritative unless it errors.

## 2026-02-05 (Preferred Exchange Historical Close)

### Decision/Learning
Historical close lookup now stops after a successful preferred exchange response.

### Reason
Fallbacks on non-error responses multiply HTTP calls during change-rate calculations.

### Impact
Preferred exchange is authoritative for historical closes unless it errors.

## 2026-02-05 (Dashboard TTL Cache)

### Decision/Learning
Main dashboard summary is cached for a short TTL to avoid recomputing on every loop iteration.

### Reason
Repeated summary builds trigger multiple DB/API calls and slow down menu navigation.

### Impact
Summary refresh is time-based; set TTL with `_SUMMARY_CACHE_TTL_SECONDS` in `cli.main`.

## 2026-02-06 (Edit Prompt Consistency)

### Decision/Learning
All update flows now treat blank/whitespace input as "keep current value", and deposit note uses `/clear` for explicit deletion.

### Reason
Mixed edit behaviors caused accidental empty updates and inconsistent UX across menus.

### Impact
Keep Enter-as-retain as the default rule for future edit prompts; use explicit clear tokens for nullable text fields.

## 2026-02-06 (KIS Account Sync)

### Decision/Learning
`KisDomesticBalanceClient`와 `KisAccountSyncService`를 추가해 KIS 계좌 예수금/보유수량을 내부 `accounts`/`holdings`로 동기화한다.

### Reason
수동 예수금/보유수량 입력은 실제 계좌 상태와 빠르게 어긋나므로, KIS 잔고 API를 단일 소스로 반영할 필요가 있다.

### Impact
`.env`에 `KIS_CANO`/`KIS_ACNT_PRDT_CD`(또는 `KIS_ACCOUNT_NO`)를 설정하면 계좌 메뉴의 `Sync KIS account`로 동기화 가능하며, 신규 티커는 `KIS 자동동기화` 그룹에 자동 생성된다.

## 2026-02-07 (KIS Sync Safety)

### Decision/Learning
`KisAccountSyncService`는 계좌 보유내역 동기화 시 `delete_by_account` 전체 삭제 대신 stock 단위 diff(create/update/delete)로 반영한다.

### Reason
전체 삭제 후 재생성 방식은 중간 실패 시 보유내역 유실 위험이 있어 데이터 무결성에 취약하다.

### Impact
향후 동기화 로직은 destructive reset을 피하고, 기존 데이터 대비 변경분만 적용하는 패턴을 유지한다.

## 2026-02-07 (Domestic Order TR Mapping)

### Decision/Learning
`KisDomesticOrderClient` 주문 엔드포인트를 `/uapi/domestic-stock/v1/trading/order-cash`로 고정하고, TR ID는 `buy=TTTC0012U`, `sell=TTTC0011U`(demo는 `V*`)로 매핑한다.

### Reason
KIS 주문 API는 action/env 조합마다 TR ID가 달라 잘못 매핑하면 정상 응답 없이 주문이 실패한다.

### Impact
국내 주문 기능 확장 시 side/env 기반 TR ID 선택을 공통 규칙으로 재사용하고, 회귀 테스트에서 엔드포인트와 TR ID를 함께 검증한다.

## 2026-02-09 (Overseas Order TR Mapping)

### Decision/Learning
`KisOverseasOrderClient`를 추가하고 미국 주문 TR ID를 `buy=TTTT1002U`, `sell=TTTT1006U`(demo는 `V*`)로 매핑했다.

### Reason
해외 주문도 action/env 조합별 TR ID가 다르므로 매핑이 틀리면 API 호출이 성공하지 않는다.

### Impact
해외 주문 기능 구현 시 side/env 기반 TR ID 선택을 단일 클라이언트 규칙으로 재사용하고, 회귀 테스트에서 buy/sell 매핑을 함께 검증한다.

## 2026-02-09 (Order Token Auto-Refresh)

### Decision/Learning
`KisDomesticOrderClient`와 `KisOverseasOrderClient`에 `token_manager` 기반 토큰 만료 자동 재시도(1회)를 추가했다.

### Reason
주문 API도 가격/잔고 API와 동일하게 `EGW00123`가 발생할 수 있어, 수동 재시도 없이 주문 흐름을 복구해야 한다.

### Impact
주문 클라이언트 초기화 시 `token_manager`를 주입하면 만료 토큰에서 자동으로 새 토큰을 받아 동일 요청을 1회 재시도한다.
