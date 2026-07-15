# Runbook

빌드/테스트/실행 명령어 모음. 웹 UI/Docker는 없음 — `cmd/pm` CLI를 Claude Code 스킬
(portfolio-data, portfolio-sync)이나 직접 `go run`으로 호출.

## 설치

```bash
make go-tools              # sqlc 설치 (go install)
```

## 사용

```bash
go run ./cmd/pm help                    # 서브커맨드 목록
go run ./cmd/pm account list            # 예: 계좌 목록
go run ./cmd/pm dashboard               # 포트폴리오 요약
```

`cmd/pm`의 전체 리소스/verb 표는 `go run ./cmd/pm help` 출력 또는
`.claude/skills/portfolio-data/SKILL.md` 참고. 가격 동기화·계좌 동기화는 더 이상
백그라운드로 자동 실행되지 않음 — `pm price-sync`/`pm sync`를 필요할 때 직접 호출.

## 코드 생성

```bash
make go-gen                # sqlc generate
```

## 빌드

```bash
make go-build              # go build ./...
go build -o pm ./cmd/pm    # pm 바이너리만
```

## 테스트

```bash
make go-test               # go test ./... (integration 제외)
make go-cover              # 커버리지 + 85% 게이트 (생성 코드 제외)
go test -tags integration ./...   # 통합 테스트 포함
go test ./internal/arch/   # 아키텍처 가드 (레이어 경계)
```

### rebalance-plan / execute-rebalance-plan 스킬 스크립트 (Python)

`.claude/skills/{rebalance-plan,execute-rebalance-plan}/scripts/*.py`는 Go 빌드에 포함되지 않는
독립 유틸리티 — 표준 라이브러리 `unittest`로 커버:

```bash
python3 -m unittest discover -s .claude/skills/rebalance-plan/scripts -p 'test_*.py'
python3 -m unittest discover -s .claude/skills/execute-rebalance-plan/scripts -p 'test_*.py'
```

### Live API smoke tests

Live tests are opt-in and must never run in CI by default.

```bash
set -a && source .env && set +a
TOSS_LIVE=1 go test ./internal/toss -run TestLiveFetchAccountSnapshot -count=1 -v
```

Toss account sync requires:

- `TOSS_CLIENT_ID`
- `TOSS_CLIENT_SECRET`
- `TOSS_BASE_URL` (optional; defaults to `https://openapi.tossinvest.com`)
- `TOSS_ACCOUNT_SEQ` (optional for the live test; when absent, the first account from `/api/v1/accounts` is used)

The live test calls only read endpoints: OAuth token issuance, accounts, holdings, KRW/USD buying power, and USD/KRW exchange rate. It logs counts and presence checks only; do not print tokens, account numbers, raw holdings, or balances.

## 린트/검증

```bash
make go-vet                # go vet ./...
make go-lint                # golangci-lint run
sqlc diff                   # sqlc 생성물 최신 여부
make go-check                # generate + build + vet + lint + test 일괄
```

## 디버깅

### KST 마이그레이션

기존 DB 행은 `+00:00` offset 으로 저장되어 있으나, 신규 행은 `+09:00`(KST)로 저장됨.
코드 레벨에서 파싱 시 양쪽 모두 허용 (`internal/ktime` 패키지). 자연 복구되며 별도 마이그레이션 불필요.

읽기 영향 모델: Stock, Group, Account, Holding, Deposit, StockPrice (6개).
`OrderExecution`은 예외 (KST 도입 이후 생성). 타임스탬프는 `ktime.Now()`로 설정.

### Pre-commit `go test` build failure on partial commits

Pre-commit stashes **unstaged tracked** changes but leaves **untracked** files
in place. An untracked test file referencing unstaged implementation code makes
`go test ./...` fail at build time during commit. Fix: stage the implementation
together with its test, or move the untracked test file aside for that commit.
