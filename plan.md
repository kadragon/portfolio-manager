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
- [ ] `KisBaseClient` 추상 클래스 도입
- [ ] 공통 헤더 처리, 환경 변수 기반 TR ID 매핑 로직 통합

### 2.2 시장 감지 로직 개선
- [ ] `KisUnifiedPriceClient`의 티커 길이 기반 감지 로직을 보다 명확한 유틸리티로 분리

---

## 3. 테스트 및 검증
- [ ] 각 리팩토링 단계 후 `pytest` 실행
- [ ] CLI 정상 동작 확인 (통합 테스트)

---

## 기존 체크리스트 (참고용)

### MVP 완료 항목
- [x] KIS 인증 및 토큰 관리
- [x] 국내/해외 주식 시세 조회
- [x] 계좌/그룹/보유종목 CRUD
- [x] 통합 대시보드 출력

### Next Action ("go" 시 시작)
1. **CLI 모듈 파일명 변경 및 import 수정**
2. **Services 폴더 구조화**
