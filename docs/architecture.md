# Architecture

## 레이어 구조

```
cmd/pm, cmd/rebalance-order   CLI 엔트리포인트 — Claude Code 스킬이 호출 (portfolio-data,
                               portfolio-sync, execute-rebalance-plan). 상시 구동 서버 없음,
                               서브커맨드 하나당 프로세스 하나로 부팅·종료
  ↓
internal/container/           합성 루트 — cmd/가 여기서 repository/service/KIS 클라이언트를 얻음
  ↓
internal/services/            비즈니스 로직 (포트폴리오, 가격, 리밸런싱, KIS 동기화)
  ↓
internal/repositories/        DB 액세스 — GP-1 (sqlc 래핑)
  ↓
internal/db/sqlc/             생성된 쿼리 (modernc.org/sqlite, pure Go)
```

레이어 의존성 방향: `cmd/ → services/ → repositories/ → db/sqlc`. `cmd/`는 `container` 경유로만
service/repository에 접근 — `internal/db`·`internal/db/sqlc` 직접 import 금지.

웹 UI(Echo+HTMX)와 Docker 배포는 걷어냈다 — 포트폴리오 조작은 전부 `cmd/pm` 서브커맨드(계좌/그룹/
종목/입금/보유 CRUD, KIS/Toss 동기화, 가격 동기화, 대시보드)와 이를 감싼 Claude Code 스킬로 이뤄진다.
가격 동기화·밴드 알림은 더 이상 백그라운드로 자동 실행되지 않음 — 전부 온디맨드(`pm price-sync`
호출 시에만).

## 핵심 원칙

1. **Repository 레이어가 모든 DB 액세스 소유** (GP-1) — `internal/arch/arch_test.go`가 강제
2. **레이어 의존성 역전 금지** (GP-3) — `internal/arch/arch_test.go`가 강제
3. **KIS live 테스트는 `KIS_LIVE=1` 가드** (GP-2) — 미설정 시 `t.Skip`
4. **시크릿은 `.env`로만** (GP-4) — golangci-lint(gosec)가 검사

## 주요 패키지

| 패키지 | 역할 |
|--------|------|
| `internal/container` | 합성 루트 (composition root). DB 위에 repository/service/KIS 클라이언트 조립 |
| `internal/models` | 도메인 구조체 |
| `internal/{uuidx,numeric,datex,ktime}` | Peewee 호환 SQLite 타입 (UUID hex, NUMERIC affinity, KST datetime) |
| `internal/kis` | KIS API 클라이언트 (auth/token/price/balance/order, 통합 클라이언트) — **주의:** `DomesticOrderClient`/`OverseasOrderClient`는 CANO/AcntPrdtCd를 struct 필드로 굽는다(계좌별로 매번 새 client 필요). `DomesticBalanceClient.FetchAccountSnapshot(cano, acntPrdtCd string)`처럼 메서드 인자로 받지 않으니 재사용하면 다른 계좌 주문이 잘못된 계좌로 나간다 — 계좌마다 `container.BuildKISOrderClient(keyID, cano, acntPrdtCd)`로 새로 만들 것 (`internal/services/order_execution_service.go` 참고) |
| `cmd/pm` | CLI — 리소스(`account`/`group`/`stock`/`deposit`/`holding`/`sync`/`classify-stocks`/`dashboard`/`price-sync`)별 서브커맨드, JSON stdout |

## 데이터 모델

SQLite 단일 파일 (`.data/portfolio.db`). 스키마는 `internal/db/schema.sql` (Peewee 프로덕션 DB와 컬럼 순서/인덱스명 호환). 주요 테이블: `groupmodel`, `stockmodel`, `accountmodel`, `holdingmodel`, `depositmodel`, `stockpricemodel`, `orderexecutionmodel`.

## DB 접근 규칙

- 모든 쿼리는 Repository 메서드를 통해서만 (sqlc 쿼리 직접 호출 금지)
- `cmd/`·`services/`에서 sqlc 직접 호출 금지 (arch 테스트로 강제)
- 쿼리 추가: `query/queries.sql` 편집 → `make go-gen` → `internal/db/sqlc/` 생성 확인
- 타임스탬프는 Go 코드(`ktime.Now()`)에서 설정 — SQL DEFAULT 미사용 (Peewee 호환)
