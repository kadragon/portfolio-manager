# Runbook

빌드/테스트/실행/배포 명령어 모음.

## 설치

```bash
make go-tools              # sqlc + templ 설치 (go install)
make setup                 # Tailwind + DaisyUI standalone CLI
```

## 개발

```bash
make dev                   # 웹 + CSS watch
go run ./cmd/portfolio-web # 웹만 (http://127.0.0.1:8000)
make css-watch             # CSS만
```

## 코드 생성

```bash
make go-gen                # sqlc generate + templ generate
```

## 빌드

```bash
make go-build              # go build ./...
go build -trimpath -ldflags="-s -w" -o portfolio-web ./cmd/portfolio-web
```

## 테스트

```bash
make go-test               # go test ./... (integration 제외)
make go-cover              # 커버리지 + 85% 게이트 (생성 코드 제외)
go test -tags integration ./...   # 통합 테스트 포함
go test ./internal/arch/   # 아키텍처 가드 (레이어 경계)
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
make go-lint               # golangci-lint run
templ generate --check     # 템플릿 생성물 최신 여부
sqlc diff                  # sqlc 생성물 최신 여부
make go-check              # build + vet + lint + test 일괄
```

## Docker

(docs/docker.md 참조)

## 배포

멀티스테이지 빌드 (`Dockerfile`): `golang:alpine`에서 `CGO_ENABLED=0` 정적 바이너리 →
`alpine` 런타임 (ca-certificates, tzdata). `.data` 볼륨에 SQLite DB 영속.

## 디버깅

### KST 마이그레이션

기존 DB 행은 `+00:00` offset 으로 저장되어 있으나, 신규 행은 `+09:00`(KST)로 저장됨.
코드 레벨에서 파싱 시 양쪽 모두 허용 (`internal/ktime` 패키지). 자연 복구되며 별도 마이그레이션 불필요.

읽기 영향 모델: Stock, Group, Account, Holding, Deposit, StockPrice (6개).
`OrderExecution`은 예외 (KST 도입 이후 생성). 타임스탬프는 `ktime.Now()`로 설정.
