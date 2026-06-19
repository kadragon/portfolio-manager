# Architecture

## 레이어 구조

```
cmd/portfolio-web/        엔트리포인트 (Echo 서버, graceful shutdown, /static)
  ↓
internal/web/handlers/    Echo 핸들러 (HTMX 뷰)
  ↓
internal/services/        비즈니스 로직 (포트폴리오, 가격, 리밸런싱, KIS 동기화)
  ↓
internal/repositories/    DB 액세스 — GP-1 (sqlc 래핑)
  ↓
internal/db/sqlc/         생성된 쿼리 (modernc.org/sqlite, pure Go)
```

레이어 의존성 방향: `web/ → services/ → repositories/ → db/sqlc`

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
| `internal/kis` | KIS API 클라이언트 (auth/token/price/balance/order, 통합 클라이언트) |
| `internal/web/templates` | templ 템플릿 + 헬퍼 |
| `internal/web/format` | Jinja 필터 대응 포매터 |

## 데이터 모델

SQLite 단일 파일 (`.data/portfolio.db`). 스키마는 `internal/db/schema.sql` (Peewee 프로덕션 DB와 컬럼 순서/인덱스명 호환). 주요 테이블: `groupmodel`, `stockmodel`, `accountmodel`, `holdingmodel`, `depositmodel`, `stockpricemodel`, `orderexecutionmodel`.

## DB 접근 규칙

- 모든 쿼리는 Repository 메서드를 통해서만 (sqlc 쿼리 직접 호출 금지)
- `web/`·`services/`에서 sqlc 직접 호출 금지 (arch 테스트로 강제)
- 쿼리 추가: `query/queries.sql` 편집 → `make go-gen` → `internal/db/sqlc/` 생성 확인
- 타임스탬프는 Go 코드(`ktime.Now()`)에서 설정 — SQL DEFAULT 미사용 (Peewee 호환)
