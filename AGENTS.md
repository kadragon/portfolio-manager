# Portfolio Manager Agent Rules

Go CLI(`cmd/pm`) + Claude Code 스킬로 제어하는 포트폴리오 관리 도구. 웹 UI/Docker 없음. SQLite
(modernc pure-Go) + sqlc, KIS Open Trading API/Toss Invest API 연동.

## Docs Index (read on demand)

| File | When to read |
|------|-------------|
| `docs/architecture.md` | 소스 구조 수정, 새 모듈 추가, 레이어 경계 관련 작업 전 |
| `docs/conventions.md` | 새 파일 작성, CLI 서브커맨드 추가, 커밋 메시지 작성 전 |
| `docs/runbook.md` | 빌드/테스트 명령어, live API smoke, 실패 디버깅 시 |
| `docs/workflows.md` | 구현 사이클 시작 시, 컨텍스트 관리 전략 필요 시 |
| `docs/delegation.md` | 서브에이전트 위임 패턴, spawn 계약, 라우팅 테이블 |
| `docs/eval-criteria.md` | 평가 기준, Sprint Contract, 완료 정의 |

## Golden Principles

Invariants. (1)(3) enforced by `internal/arch/arch_test.go`. (4) by golangci-lint (gosec) in pre-commit + CI. (2)(5) convention only.

1. **Repository layer owns all DB access** — `cmd/` (outside `internal/container`) and `services/` must not import `internal/db` (or `internal/db/sqlc`). All DB access goes through `*Repository` fields on `container.Container` (`Accounts`, `Stocks`, `Holdings`, …). Enforced by `TestCmdHasNoDirectDBAccess` / `TestServicesHaveNoDirectDBAccess` in `internal/arch/arch_test.go`.
   ```go
   // correct — repository injected via container
   account, err := c.Accounts.GetByID(ctx, id)
   // violation — blocked by arch test (cmd/ or services/ importing the DB layer)
   import "github.com/kadragon/portfolio-manager/internal/db/sqlc"
   ```

2. **KIS live tests guard with `KIS_LIVE=1`** — e.g. `internal/kis/overseas_price_live_test.go`. The test calls `t.Skip` unless `KIS_LIVE=1` is set; CI never sets it. Token issuance rate-limited to 1/min.

3. **Layer dependency direction** — `cmd → internal/services → internal/repositories → internal/db`, via `internal/container` as composition root. Reverse imports are violations. Enforced by `TestRepositoriesDoNotImportServices` in `internal/arch/arch_test.go`.

4. **Secrets via `.env` only** — No hardcoded credentials, keys, or tokens in source. Enforced by gosec (via golangci-lint) in pre-commit + CI.

5. **Commit messages use `[TYPE]` prefix** — `[FEAT]` · `[FIX]` · `[REFACTOR]` · `[TEST]` · `[DOCS]` · `[CONSTRAINT]` · `[HARNESS]` · `[PLAN]`. Convention — not yet mechanically enforced.

## Delegation

See `docs/delegation.md` — hard stops, spawn contract, routing table.

## Context Management

- 대형 작업 시작 시 `handoff-{feature}.md` 작성 (컨텍스트가 신선할 때).
- 컨텍스트가 채워지면 인플레이스 압축이 아닌 **리셋 + 핸드오프 파일** 전략 사용.
- 징후: 앞 기능은 완전히 구현하고 뒤는 스텁만 채움 → 즉시 리셋.

## Working Notes

- **KIS 토큰 발급은 분당 1회 제한** — live auth 검증 루프 반복 금지.
- `KIS_ENV`: `real` / `demo` / `vps` / `paper`. 잘못된 값은 조용히 production URL로 라우팅됨.
- 가격 동기화·계좌 동기화는 자동 실행되지 않음 (백그라운드 작업 없음) — `cmd/pm price-sync`/`sync`를 필요할 때 직접 호출.
- **CLI 우회 감지 시 `pm` 확장 우선** — 사용자 요청에 답하려고 `sqlite3 .data/portfolio.db` 직접 쿼리나 즉석 스크립트(python 등)로 `pm` 출력을 가공했다면, 일회성으로 넘기지 말고 그 자리에서 `cmd/pm`에 read/sort 서브커맨드나 플래그를 추가해라 — 같은 우회가 다음에도 반복된다.

## Agent skills

### Issue tracker

Issues live in GitHub Issues (`kadragon/portfolio-manager`). See `docs/agents/issue-tracker.md`.

### Triage labels

Default label vocabulary (needs-triage / needs-info / ready-for-agent / ready-for-human / wontfix). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context — `CONTEXT.md` at repo root + `docs/adr/README.md`. See `docs/agents/domain.md`.

## Language Policy

- 코드, 커밋, docs/: English
- 사용자 커뮤니케이션: Korean

## Maintenance

Update AGENTS.md **only** when ALL conditions met:
1. Info not discoverable from code, config, or `docs/`
2. Operationally significant — affects build, test, deploy, or runtime safety
3. Would likely cause agent mistakes if undocumented
4. Stable — not task-specific or temporary

**Never add:** architecture summaries, directory overviews, style conventions already enforced by linter, temporary notes, task-specific instructions. Move long content to `docs/*.md` with a pointer here.

**Size target:** ≤100 lines. Hard warn >200 — split to `docs/` immediately.
