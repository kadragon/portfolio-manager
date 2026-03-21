# Portfolio Manager

FastAPI + HTMX 기반 포트폴리오 관리 웹 애플리케이션. 한국투자증권(KIS) Open Trading API와 연동하여 국내/해외 주식 보유 현황, 실시간 시세, 리밸런싱 추천을 제공합니다.

## 주요 기능

- **대시보드** — 보유 종목, 현재가, 평가액, 수익률(연환산 포함), 그룹별 비중 요약
- **그룹/종목 관리** — 그룹별 목표 비중 설정, 종목 추가/이동/삭제
- **계좌 관리** — 복수 계좌 예수금 관리, KIS 계좌 자동 동기화
- **입금 내역** — 일자별 투자원금 기록
- **리밸런싱** — 그룹/지역 비중 진단, 매매 추천, KIS API 주문 실행
- **시세 캐싱** — 일별 가격 DB 캐시 + 세션 내 메모리 캐시

## 기술 스택

| 계층 | 기술 |
|------|------|
| Backend | Python 3.13+, FastAPI, Jinja2, HTMX |
| Frontend | Tailwind CSS v4 (standalone CLI), DaisyUI v5 |
| Database | Supabase (PostgreSQL) |
| 시세 API | 한국투자증권 KIS Open Trading API |
| 환율 | 한국수출입은행 EXIM API |
| CLI | Rich (터미널 대시보드) |

## 시작하기

### 사전 요구사항

- Python 3.13+
- [uv](https://docs.astral.sh/uv/) (패키지 매니저)
- Supabase 프로젝트 (`.env`에 URL/키 설정)

### 설치

```bash
# 의존성 설치
uv sync

# Tailwind CSS + DaisyUI 설치 (Node.js 불필요)
make setup
```

### 환경 변수 (`.env`)

```env
# Supabase
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_ROLE_KEY=your-service-role-key

# KIS API (선택)
KIS_APP_KEY=your-app-key
KIS_APP_SECRET=your-app-secret
KIS_ENV=real              # real 또는 demo
KIS_CANO=12345678         # 계좌번호 (동기화/주문용)
KIS_ACNT_PRDT_CD=01       # 계좌 상품코드

# 환율 (선택)
EXIM_AUTH_KEY=your-exim-key

# Supabase 자동 resume (선택)
SUPABASE_ACCESS_TOKEN=your-personal-access-token
```

### 실행

```bash
# 웹 서버 + CSS 워치 동시 실행
make dev

# 또는 개별 실행
make css-watch &          # CSS 자동 빌드
uv run portfolio-web      # http://127.0.0.1:8000
```

### CSS 빌드

```bash
make setup      # Tailwind standalone CLI + DaisyUI 다운로드
make css-watch  # 개발 모드 (파일 변경 시 자동 빌드)
make css-build  # 프로덕션 빌드 (minified)
```

## 프로젝트 구조

```
src/portfolio_manager/
  web/                    # FastAPI 웹 앱
    app.py                # 앱 팩토리, Jinja2 필터
    routes/               # 라우트 (dashboard, groups, accounts, deposits, rebalance)
    templates/            # Jinja2 템플릿 (DaisyUI + HTMX)
    static/               # CSS (Tailwind 출력), JS, favicon
    tailwind/             # Tailwind 입력 CSS + DaisyUI 번들
  cli/                    # Rich 터미널 CLI
  models/                 # 데이터 모델
  repositories/           # Supabase 리포지토리
  services/               # 비즈니스 로직 (포트폴리오, 가격, 리밸런싱, KIS 클라이언트)
scripts/                  # 유틸리티 스크립트 (setup-tailwind.sh, KIS 검증 등)
supabase/migrations/      # DB 마이그레이션
tests/                    # pytest 테스트 (coverage 85%+)
```

## 테스트

```bash
uv run pytest              # 단위 테스트 (integration 제외)
uv run pytest -m integration  # 외부 API 연동 테스트
```

## 라이선스

Proprietary
