# Conventions

## 커밋 메시지

`[TYPE] description` 형식 (Conventional Commits `type(scope):` 사용 안 함):

- `[FEAT]` 기능 추가
- `[FIX]` 버그 수정 (재현 테스트 동반)
- `[REFACTOR]` 구조 개선 (동작 불변)
- `[TEST]` 테스트 전용 (신규 커버리지, 테스트 리팩터)
- `[DOCS]` 문서
- `[CONSTRAINT]` 구조적 가드 (lint/schema/type) — 프로덕션 코드 미변경
- `[HARNESS]` CI, 린터, 평가 기준, 툴링
- `[PLAN]` 백로그/태스크 변경

## Go 코드 스타일

- golangci-lint (gofmt/goimports, staticcheck, gosec, errorlint, revive 등) — `.golangci.yml` 참조
- `go vet` 통과 필수
- 에러는 `fmt.Errorf(..., %w)`로 래핑, `errors.Is`/`errors.As`로 검사 (errorlint)
- 외부 IO·비결정 의존성만 모킹; 그 외 통합 테스트 우선

## 네이밍

- 파일: snake_case (`stock_service.go`)
- 익스포트 식별자: PascalCase, 비익스포트: camelCase
- 패키지: 짧은 소문자 단일 단어
- 상수: 관용적 Go (PascalCase 익스포트, 비익스포트는 camelCase 허용)

## 테스트

- 표준 `testing`, `*_test.go` (대상 패키지 옆)
- 화이트박스(`package foo`) 또는 블랙박스(`package foo_test`) — 헬퍼 검증은 화이트박스
- KIS live(통합) 테스트는 `KIS_LIVE=1` 환경변수 가드 — 미설정 시 `t.Skip` (예: `internal/kis/overseas_price_live_test.go`, GP-2)
- 커버리지 85%+ 유지 (`make go-cover`; 생성 코드 제외)
- 테이블 드리븐 테스트 선호

## 코드 생성

- DB 쿼리: `query/queries.sql` → `make go-gen` → `internal/db/sqlc/` (committed)
- 템플릿: `*.templ` → `make go-gen` → `*_templ.go` (gitignored)
- 변경 후 `sqlc diff` / `templ generate --check` 통과 확인 (pre-commit/CI에서 검사)

## HTMX 패턴

- `HX-Request` 헤더로 partial/full 분기
- 템플릿: templ, DaisyUI 컴포넌트
- `internal/web/handlers/render.go` — 본문 출력 전 상태 코드 설정
