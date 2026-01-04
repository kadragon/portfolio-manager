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
