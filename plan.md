# plan.md

# 프로젝트 관리 및 리팩토링 계획

## 목표
- Python + Rich 기반의 CLI 주식관리 프로그램 고도화
- 코드 품질 개선, 모듈화 강화, 중복 제거
- TDD 및 Tidy First 원칙 준수

## 0. 현재 상태 및 분석 (2026-01-04)
- **MVP 단계 완료**: 기본적인 CRUD 및 시세 조회 기능 동작
- **구조적 문제**: 
  - `services/` 폴더에 KIS 관련 파일이 과도하게 밀집됨
  - `cli/` 폴더 내 파일명에 `rich_` 접두사가 중복 사용됨
  - `main.py`가 초기화 로직과 UI 루프를 모두 담당하여 복잡도가 높음
  - KIS 국내/해외 시세 클라이언트 간 코드 중복 존재

---

## 1. 리팩토링 계획 (Phase 1: Structural Changes)
*Behavioral 변화 없이 구조만 개선 (Tidy First)*

### 1.1 CLI 모듈 정리
- [x] `src/portfolio_manager/cli/` 내 `rich_*.py` 파일명 변경
  - `rich_accounts.py` -> `accounts.py`
  - `rich_groups.py` -> `groups.py`
  - `rich_holdings.py` -> `holdings.py`
  - `rich_stocks.py` -> `stocks.py`
  - `rich_menu.py` -> `menu.py`
  - `rich_app.py` -> `app.py`
- [x] 관련 import 문 전체 수정

### 1.2 Services 모듈 구조화
- [x] `src/portfolio_manager/services/kis/` 서브 패키지 생성 및 이동:
  - `kis_auth_client.py`, `kis_domestic_info_client.py`, `kis_domestic_price_client.py`, `kis_overseas_price_client.py`, `kis_price_parser.py`, `kis_token_manager.py`, `kis_token_store.py`, `kis_unified_price_client.py`
- [x] `src/portfolio_manager/services/exchange/` 서브 패키지 생성 및 이동:
  - `exchange_rate_service.py`, `exim_exchange_rate_client.py`
- [x] 관련 import 문 전체 수정

### 1.3 초기화 로직 분리
- [x] `src/portfolio_manager/core/container.py` (또는 factory) 생성
- [x] `main.py`의 서비스 및 리포지토리 초기화 로직을 컨테이너로 이동

---

## 2. 리팩토링 계획 (Phase 2: Behavioral/Internal Changes)
*기능적 개선 및 중복 제거*

### 2.x 대시보드 해외주식 표시 개선
- [x] 대시보드에 미국주식명, 가격이 조회되지 않은 문제 수정
- [x] 대시보드에서 해외주식 quantity 표기 시 소수점 첫번째 자리에서 반올림하여 정수로만 표기
- [x] 해외주식 이름이 비어있을 때 다른 거래소 조회로 보완
- [x] 해외주식 Value를 KRW로 환산해 표시

### 2.3 그룹별 목표 비중 설정 기능 추가
- [x] **Migration**: `groups` 테이블에 `target_percentage` 컬럼 추가 (numeric, nullable or default 0)
- [x] **Model**: `Group` 모델에 `target_percentage` 필드 추가
- [x] **Repository**: `GroupRepository`의 `create`, `update`, `list_all` 메서드 수정
- [x] **CLI**:
  - 그룹 목록 조회 시 목표 비중 표시
  - 그룹 생성/수정 시 목표 비중 입력 받기

### 2.4 투자금(원금) 관리 기능 추가
- **개요**: 계좌의 예수금(Cash Balance)과 별개로, 실제 투입된 원금을 추적하여 정확한 수익률 계산 (계좌와 무관하게 전역 관리, 일별 유니크)
- **Migration**: `deposits` 테이블 생성 및 수정
  - [x] 컬럼: `id`, `amount`, `date` (Unique), `note` (account_id 제거)
- **Model**: `Deposit` 모델 생성
  - [x] `Deposit` 모델 구현
- **Repository**: `DepositRepository` 구현 (추가, 수정, 조회, 삭제, 전체 합계)
  - [x] `DepositRepository` 구현 및 테스트
- **CLI**:
  - [x] 입금 내역 관리 메뉴 추가 (추가/수정/목록/삭제)
  - [x] 대시보드 업데이트: '총 투자 원금' 표시 및 '투자 수익률' 계산 로직 반영

### 2.1 KIS 클라이언트 추상화
- [x] `KisBaseClient` 추상 클래스 도입
- [x] 공통 헤더 처리, 환경 변수 기반 TR ID 매핑 로직 통합

### 2.5 KIS API 토큰 자동 갱신 (Token Auto-Refresh)
- [x] **Test**: 토큰 만료 에러 감지 테스트 작성
- [x] **Implementation**: 500 에러 응답에서 토큰 만료 감지 (msg_cd: 'EGW00123')
- [x] **Test**: 토큰 만료 시 자동 재시도 테스트 작성
- [x] **Implementation**: 토큰 만료 시 자동으로 재발급 및 재시도
- [x] **Integration**: `KisDomesticPriceClient`, `KisOverseasPriceClient`에 retry 로직 통합

### 2.2 시장 감지 로직 개선
- [x] `KisUnifiedPriceClient`의 티커 길이 기반 감지 로직을 보다 명확한 유틸리티로 분리

### 2.6 리밸런싱 추천 기능 (Rebalancing Recommendations)
- **개요**: 그룹 목표 비중과 현재 평가액 차이를 기준으로 개별 주식 매매 추천
  - 매도 우선순위: 해외주식 먼저
  - 매수 우선순위: 국내주식 먼저
- **Model**: `RebalanceRecommendation` 데이터 클래스 생성
  - [x] Test: 추천 모델이 ticker, action(buy/sell), amount, priority 포함
  - [x] Impl: `RebalanceRecommendation` 구현
- **Service**: `RebalanceService` 구현
  - [x] Test: 그룹별 현재 평가액과 목표 차이 계산
  - [x] Impl: 그룹별 차이 계산 로직
  - [x] Test: 매도 추천 시 해외주식(USD) 우선
  - [x] Impl: 해외주식 우선 매도 로직
  - [x] Test: 매수 추천 시 국내주식(KRW) 우선
  - [x] Impl: 국내주식 우선 매수 로직
- **CLI**: 리밸런싱 메뉴 추가
  - [x] Test: 메인 메뉴에 rebalance 옵션 추가
  - [x] Impl: choose_main_menu에 rebalance 옵션
  - [x] Test: 리밸런싱 추천 테이블 렌더링
- [x] Impl: render_rebalance_recommendations 함수
  - [x] Test: 리밸런싱 추천 표에 매도/매수 계좌를 동일 계좌로 표시

### 2.7 주식별 과거 대비 변동률 표시 (1Y/6M/1M)
- [x] **Test**: PriceService가 1Y/6M/1M 변동률을 계산한다
- [x] **Implementation**: PriceService에 변동률 계산 메서드 추가
- [x] **Test**: PortfolioService가 보유주식에 변동률을 포함한다
- [x] **Implementation**: PortfolioService가 PriceService에서 변동률을 받아 StockHoldingWithPrice에 저장
- [x] **Test**: 대시보드에 1Y/6M/1M 컬럼을 표시한다
- [x] **Implementation**: render_dashboard에 1Y/6M/1M 컬럼 추가
- [x] **Test**: KIS 국내/해외 과거 종가 조회 API 요청을 구성한다
- [x] **Implementation**: KisDomesticPriceClient/KisOverseasPriceClient에 과거 종가 조회 추가
- [x] **Test**: 휴장일이면 이전 영업일로 자동 보정한다
- [x] **Implementation**: 과거 종가 조회 시 이전 영업일 탐색 로직 추가

### 2.8 해외 거래소 캐시 (NAS/NYS/AMS)
- [x] **Migration**: `stocks` 테이블에 `exchange` 컬럼 추가 (text, nullable)
- [x] **Model/Repository**: `Stock` 모델에 `exchange` 필드 추가 및 `StockRepository.update_exchange()` 구현
- [x] **Test**: StockRepository가 exchange를 업데이트한다
- [x] **Test**: KisUnifiedPriceClient가 저장된 거래소를 우선 조회하고 실패 시 다음 거래소로 넘어간다
- [x] **Test**: KisUnifiedPriceClient의 과거 종가 조회도 저장된 거래소를 우선 조회한다
- [x] **Test**: PortfolioService가 조회 성공한 거래소로 stocks.exchange를 갱신한다
- [x] **Implementation**: PriceService가 preferred_exchange를 전달하고 exchange를 반환한다
- [x] **Implementation**: PortfolioService가 exchange 캐시를 갱신한다

### 2.9 포트폴리오 정렬 및 국내 1Y 수익률 수정
- [x] **Test**: PortfolioService가 holdings를 value_krw 기준 내림차순 정렬한다
- [x] **Implementation**: get_portfolio_summary에서 return 전 holdings 정렬
- [x] **Test**: 그룹 요약 테이블이 총 평가액 기준 내림차순 정렬된다
- [x] **Implementation**: app.py에서 group_totals를 정렬하여 출력
- [x] **Test**: KisDomesticPriceClient.fetch_historical_close가 휴장일에도 직전 영업일 종가를 반환한다
- [x] **Implementation**: fetch_historical_close에 date range + fallback 로직 추가

### 2.10 KIS API 인메모리 캐싱
- [x] **Test**: PriceService.get_stock_price가 메모리 캐시를 사용한다
- [x] **Implementation**: PriceService에 `_price_cache` 딕셔너리 추가
- [x] **Test**: PriceService.get_stock_change_rates가 메모리 캐시를 사용한다
- [x] **Implementation**: PriceService에 `_change_rates_cache` 딕셔너리 추가
- [x] **Test**: ExchangeRateService.get_usd_krw_rate가 메모리 캐시를 사용한다
- [x] **Implementation**: ExchangeRateService를 일반 클래스로 변환하고 `_cached_rate` 추가

---

## 3. 테스트 및 검증
- [x] 각 리팩토링 단계 후 `pytest` 실행
- [x] CLI 정상 동작 확인 (통합 테스트)

---

## 기존 체크리스트 (참고용)

### MVP 완료 항목
- [x] KIS 인증 및 토큰 관리
- [x] 국내/해외 주식 시세 조회
- [x] 계좌/그룹/보유종목 CRUD
- [x] 통합 대시보드 출력

### Next Action ("go" 시 시작)
1. **5.3.1 첫 번째 미체크 항목부터 Red→Green→Refactor 순서로 진행**
2. **각 항목 완료 시 관련 테스트만 우선 실행 후, 주기적으로 전체 테스트를 회귀 실행**

---

## 4. Performance Regression Fixes (2026-02-05)
- [x] PortfolioService에서 변동률 조회를 선택적으로 비활성화할 수 있다 (include_change_rates 플래그 추가)
- [x] 메인 대시보드 렌더링은 기본적으로 변동률 조회를 건너뛴다
- [x] PortfolioService가 전체 stocks 목록을 1회 조회해 N+1을 제거한다
- [x] HoldingRepository가 서버 사이드 집계를 사용해 보유 수량을 합산한다
- [x] KisUnifiedPriceClient 해외 현재가 조회는 선호 거래소 실패 시에만 fallback 한다
- [x] KisUnifiedPriceClient 해외 과거 종가는 선호 거래소 실패 시에만 fallback 한다
- [x] 메인 루프 대시보드 렌더링에 TTL 캐시를 적용한다

---

## 5. Rebalance Execution via KIS Order API (2026-02-07)

### 5.0 목표/범위
- 리밸런싱 추천(조회 전용)을 실제 주문(매도/매수)으로 실행한다.
- 1차 범위: 국내(KRX) + 미국(NASD/NYSE/AMEX) 주문 지원.
- 기본 동작: Dry-run(미리보기) 후 최종 확인 시에만 실제 주문 실행.
- 실행 순서: Sell 먼저, Buy 나중.
- 실행 직후 KIS 계좌 동기화(`KisAccountSyncService`)로 내부 상태를 확정한다.

### 5.1 사전 제약/결정사항
- KIS 공식 샘플 기준 주문 API/TR ID 사용:
  - 국내: `/uapi/domestic-stock/v1/trading/order-cash`
    - 매수 TR ID: `TTTC0012U` (demo: `VTTC0012U`)
    - 매도 TR ID: `TTTC0011U` (demo: `VTTC0011U`)
  - 해외: `/uapi/overseas-stock/v1/trading/order`
    - 미국 매수 TR ID: `TTTT1002U` (demo: `VTTT1002U`)
    - 미국 매도 TR ID: `TTTT1006U` (demo: `VTTT1006U`)
- 주문 본문 key는 KIS 규칙에 맞춰 대문자 필드를 유지한다.
- 토큰 만료(`msg_cd=EGW00123`)는 기존 가격조회/잔고조회와 동일하게 자동 재시도한다.

### 5.2 설계 개요 (체크박스 아님)
- 신규 서비스/클라이언트:
  - `KisDomesticOrderClient`
  - `KisOverseasOrderClient`
  - `KisUnifiedOrderClient` (ticker/market routing)
  - `RebalanceExecutionService` (추천 -> 주문 의도 생성, 실행 orchestration)
- 신규 데이터 모델:
  - `OrderIntent` (주문 전송 전 표준화 요청 모델)
  - `OrderExecutionResult` (성공/실패/스킵 결과)
  - `OrderExecutionRecord` (DB 저장용)
- 신규 저장소/스키마:
  - `order_executions` 테이블 + `OrderExecutionRepository`
  - 저장 항목: action, ticker, market, exchange, quantity, price, currency, tr_id, rt_cd, msg_cd, msg1, raw_response, created_at
- CLI:
  - Rebalance 화면에 `Preview only` / `Execute orders` 분기 추가
  - 주문 실행 전 최종 확인 prompt 추가
  - 실행 결과 요약(성공/실패/스킵) 표 출력

### 5.3 TDD 체크리스트 (다음 세션부터 순서대로 진행)

#### 5.3.1 주문 클라이언트 계약 고정
- [x] Test: `KisDomesticOrderClient`가 real env 매수 시 `TTTC0012U`와 `/uapi/domestic-stock/v1/trading/order-cash`를 사용한다
- [x] Test: `KisDomesticOrderClient`가 real env 매도 시 `TTTC0011U`를 사용한다
- [x] Test: `KisDomesticOrderClient`가 demo env에서 `VTTC0012U`/`VTTC0011U`로 변환한다
- [x] Test: `KisDomesticOrderClient`가 주문 body를 대문자 key(`CANO`, `ACNT_PRDT_CD`, `PDNO`, `ORD_DVSN`, `ORD_QTY`, `ORD_UNPR`, `EXCG_ID_DVSN_CD`)로 전송한다
- [x] Test: `KisOverseasOrderClient`가 미국 매수 시 `TTTT1002U`, 매도 시 `TTTT1006U`를 사용한다
- [x] Test: `KisOverseasOrderClient`가 demo env에서 `VTTT1002U`/`VTTT1006U`로 변환한다
- [x] Test: `KisOverseasOrderClient`가 `/uapi/overseas-stock/v1/trading/order`와 필수 body key(`OVRS_EXCG_CD`, `PDNO`, `ORD_QTY`, `OVRS_ORD_UNPR`, `ORD_DVSN`)를 전송한다
- [x] Test: 두 주문 클라이언트 모두 토큰 만료 응답(`EGW00123`) 시 1회 토큰 재발급 후 재시도한다

#### 5.3.2 주문 의도 생성/정규화
- [x] Test: `RebalanceExecutionService`가 리밸런싱 추천에서 `OrderIntent` 목록을 생성한다
- [x] Test: 수량이 소수면 주문 전 정수 수량으로 정규화한다(기본: floor)
- [x] Test: 정규화 결과 0주가 되면 해당 주문을 `skipped`로 분류한다
- [x] Test: Sell intent가 Buy intent보다 먼저 실행 순서에 배치된다
- [x] Test: 해외 주문 intent는 stock.exchange 우선, 없으면 기본 거래소 우선순위(NASD->NYSE->AMEX)를 사용한다

#### 5.3.3 실행/로깅/동기화
- [x] Test: `execute_rebalance_orders(dry_run=True)`는 API 호출 없이 intent 요약만 반환한다
- [x] Test: `execute_rebalance_orders(dry_run=False)`는 intent 순서대로 주문 API를 호출한다
- [x] Test: 주문 API 실패 1건이 있어도 나머지 주문은 계속 실행하고 개별 실패를 수집한다
- [x] Test: 실행 결과가 `order_executions`에 저장된다(성공/실패/스킵 모두 기록)
- [x] Test: 실제 실행 완료 후 `KisAccountSyncService.sync_account()`가 1회 호출된다
- [x] Test: 동기화 실패 시 주문 결과는 유지하고 동기화 실패를 별도 경고로 노출한다

#### 5.3.4 CLI 플로우
- [x] Test: 리밸런싱 메뉴가 `Preview only`/`Execute orders` 선택지를 표시한다
- [x] Test: Execute 선택 시 최종 확인 prompt에서 취소하면 실제 주문이 발생하지 않는다
- [x] Test: Execute 확정 시 실행 결과 표(성공/실패/스킵 수치)가 출력된다
- [x] Test: 실패 항목이 있을 때 ticker/action/msg_cd/msg1이 결과 화면에 표시된다

#### 5.3.5 마이그레이션/리포지토리
- [x] Test: `OrderExecutionRepository.create()`가 주문 실행 1건을 저장하고 역직렬화한다
- [x] Test: `OrderExecutionRepository.list_recent()`가 최신순으로 결과를 조회한다
- [x] Test: `order_executions` 마이그레이션이 unique 제약 없이 다건 이력 저장을 허용한다

### 5.4 구현 시 참고 규칙 (체크박스 아님)
- `RebalanceRecommendation.amount`는 표시용으로 유지하고, 실제 주문 수량 계산은 `quantity` 기준으로 처리한다.
- `quantity`가 비어 있으면 실행 전 가격을 다시 조회해 안전하게 재계산한다(재계산 실패 시 skip).
- 주문가 정책:
  - v1 기본은 지정가(`ORD_DVSN=00`) + 현재가 사용.
  - 향후 시장가/장전장후 옵션은 별도 phase로 분리.
- 결과 raw JSON은 audit 목적상 `order_executions.raw_response`에 저장한다.

### 5.5 완료 기준 (Acceptance)
- CLI에서 추천 조회 -> Preview -> 실제 주문 실행 -> 결과 확인까지 한 흐름으로 동작한다.
- 주문 실패가 일부 발생해도 전체 프로세스가 중단되지 않고 결과가 누락 없이 표시된다.
- 실행 이력이 DB에 남아 추적 가능하다.
- 실행 후 KIS 동기화를 통해 holdings/cash가 최신 상태로 반영된다.
