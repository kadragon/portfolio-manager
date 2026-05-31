# Portfolio Manager

Echo + HTMX 기반 포트폴리오 관리 웹 애플리케이션. 한국투자증권(KIS) Open Trading API와 연동하여 국내/해외 주식 보유 현황, 실시간 시세, 리밸런싱 추천을 제공합니다.

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
| Backend | Go 1.26, Echo v4 |
| Templating | templ (compiled Go templates) + HTMX |
| Frontend | Tailwind CSS v4 (standalone CLI), DaisyUI v5 |
| Database | SQLite (modernc.org/sqlite, pure Go, `CGO_ENABLED=0`) + sqlc |
| 시세 API | 한국투자증권 KIS Open Trading API |
| 환율 | 한국수출입은행 EXIM API |

## 시작하기

### 사전 요구사항

- Go 1.26+
- `.env` 파일 (KIS API 키, 환율 설정)

### 설치

```bash
make go-tools   # sqlc + templ 설치
make setup      # Tailwind standalone CLI + DaisyUI 다운로드
```

### 환경 변수 (`.env`)

```env
# KIS API (선택 — 없으면 시세 조회/계좌 동기화 비활성화)
KIS_APP_KEY=your-app-key
KIS_APP_SECRET=your-app-secret
KIS_ENV=real              # real / demo / vps / paper
KIS_CANO=12345678         # 계좌번호 8자리 (동기화/주문용)
KIS_ACNT_PRDT_CD=01       # 계좌 상품코드
# 또는: KIS_ACCOUNT_NO=1234567801  (10자리 → 8+2 분리)

# 환율 (선택 — 없으면 해외 주식 평가액 계산 불가)
EXIM_AUTH_KEY=your-exim-key
# 또는 고정 환율: USD_KRW_RATE=1350.00
```

### 실행

```bash
make dev                   # 웹 서버 + CSS 워치 동시 실행
# 또는 개별 실행
make css-watch &           # CSS 자동 빌드
go run ./cmd/portfolio-web # http://127.0.0.1:8000
```

### 코드 생성 / CSS 빌드

```bash
make go-gen     # sqlc generate + templ generate
make css-build  # 프로덕션 CSS (minified)
```

## 프로젝트 구조

```
cmd/portfolio-web/      엔트리포인트 (Echo 서버)
internal/
  web/handlers/         Echo 핸들러 (HTMX 뷰)
  web/templates/        templ 템플릿
  services/             비즈니스 로직 (포트폴리오, 가격, 리밸런싱, KIS 동기화)
  repositories/         DB 액세스 (sqlc 래핑)
  db/                   스키마 + sqlc 생성물
  kis/                  KIS API 클라이언트
  {uuidx,numeric,datex,ktime}/   SQLite 호환 타입
query/queries.sql       sqlc 쿼리 소스
```

## 테스트

```bash
make go-test               # 단위 테스트
make go-cover              # 커버리지 + 85% 게이트
go test -tags integration ./...   # 외부 API 연동 테스트
```

## 라이선스

Proprietary
