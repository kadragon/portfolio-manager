# Portfolio Manager

Claude Code 스킬로 제어하는 개인 포트폴리오 관리 CLI 도구. 한국투자증권(KIS) Open Trading API,
Toss Invest API와 연동하여 국내/해외 주식 보유 현황, 실시간 시세, 리밸런싱 추천을 제공합니다.
상시 구동되는 웹 서버는 없습니다 — 모든 조작은 `cmd/pm` CLI와 이를 감싼 Claude Code 스킬을
통해 대화형으로 이뤄집니다.

## 주요 기능

- **대시보드** — 보유 종목, 현재가, 평가액, 수익률(연환산 포함), 그룹별 비중 요약 (`portfolio-data` 스킬)
- **그룹/종목/계좌/입금/보유 관리** — CRUD 전부 `portfolio-data` 스킬
- **계좌 동기화** — KIS/Toss 계좌 동기화, 가격 갱신, 종목 자산분류를 온디맨드로 실행 (`portfolio-sync` 스킬)
- **리밸런싱** — 계획 수립은 `rebalance-plan` 스킬, TOSS/ISA 주문 실행은 `execute-rebalance-plan` 스킬 + `cmd/rebalance-order`
- **시세 캐싱** — 일별 가격 DB 캐시

## 기술 스택

| 계층 | 기술 |
|------|------|
| CLI | Go 1.26 (`cmd/pm`, `cmd/rebalance-order`) |
| 제어 인터페이스 | Claude Code 스킬 (`.claude/skills/`) |
| Database | SQLite (modernc.org/sqlite, pure Go, `CGO_ENABLED=0`) + sqlc |
| 시세 API | 한국투자증권 KIS Open Trading API |
| 환율 | 한국수출입은행 EXIM API |

## 시작하기

### 사전 요구사항

- Go 1.26+
- `.env` 파일 (KIS API 키, 환율 설정)
- Claude Code (스킬로 조작하려면)

### 설치

```bash
make go-tools   # sqlc 설치
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

# Toss Invest API (선택 — 없으면 Toss 계좌 동기화/주문 비활성화)
TOSS_CLIENT_ID=your-client-id
TOSS_CLIENT_SECRET=your-client-secret

# 환율 (선택 — 없으면 해외 주식 평가액 계산 불가)
EXIM_AUTH_KEY=your-exim-key
# 또는 고정 환율: USD_KRW_RATE=1350.00
```

### 사용

```bash
go run ./cmd/pm help          # 서브커맨드 목록
go run ./cmd/pm account list  # 예: 계좌 목록
go run ./cmd/pm dashboard     # 포트폴리오 요약
```

일반적으로는 CLI를 직접 호출하기보다 Claude Code에서 `portfolio-data`/`portfolio-sync`/
`rebalance-plan`/`execute-rebalance-plan` 스킬로 대화형 조작하는 것을 권장합니다.

### 코드 생성

```bash
make go-gen     # sqlc generate
```

## 프로젝트 구조

```
cmd/pm/                 CLI 엔트리포인트 (계좌/그룹/종목/입금/보유 CRUD, 동기화, 대시보드)
cmd/rebalance-order/    리밸런싱 주문 실행 CLI (execute-rebalance-plan 스킬이 호출)
.claude/skills/         Claude Code 스킬 (portfolio-data, portfolio-sync, rebalance-plan, ...)
internal/
  container/             합성 루트 (repository/service/KIS 클라이언트 조립)
  services/              비즈니스 로직 (포트폴리오, 가격, 리밸런싱, KIS 동기화)
  repositories/          DB 액세스 (sqlc 래핑)
  db/                    스키마 + sqlc 생성물
  kis/                   KIS API 클라이언트
  toss/                  Toss Invest API 클라이언트
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
