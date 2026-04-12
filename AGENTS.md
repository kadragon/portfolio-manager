# Portfolio Manager Agent Rules

FastAPI + HTMX 포트폴리오 관리 앱. SQLite + Peewee ORM, KIS Open Trading API 연동.

## Docs Index (read on demand)

| File | When to read |
|------|-------------|
| `docs/architecture.md` | 소스 구조 수정, 새 모듈 추가, 레이어 경계 관련 작업 전 |
| `docs/conventions.md` | 새 파일 작성, 라우트 추가, 커밋 메시지 작성 전 |
| `docs/runbook.md` | 빌드/테스트/배포 명령어 필요 시, 실패 디버깅 시 |
| `docs/workflows.md` | 구현 사이클 시작 시, 컨텍스트 관리 전략 필요 시 |
| `docs/docker.md` | Docker 실행/트러블슈팅 시 |

## Golden Principles

Invariants. (1)(3) enforced by `pytest tests/arch/`. (2)(4) enforced by pytest config + bandit. (5) convention only.

1. **Repository layer owns all DB access** — `web/` and `services/` do not call ORM query methods directly. All DB access goes through `*Repository` methods injected via `ServiceContainer`. Enforced by `tests/arch/test_layer_boundaries.py`.
   ```python
   # correct
   account = container.account_repository.get_by_id(id)
   # violation — blocked by arch test
   account = AccountModel.get_by_id(id)  # in web/ or services/
   ```

2. **KIS live tests carry `@pytest.mark.integration`** — Tests that call real KIS APIs must be marked. CI excludes them (`-m "not integration"` in addopts). Token issuance is rate-limited to 1/min.

3. **Layer dependency direction** — `web/ → services/ → repositories/ → services/database (Peewee ORM)`. Reverse imports are violations. Enforced by `tests/arch/test_layer_boundaries.py`.

4. **Secrets via `.env` only** — No hardcoded credentials, keys, or tokens in source. Enforced by bandit in pre-commit + CI.

5. **Commit messages use `[TYPE]` prefix** — `[FEAT]` · `[FIX]` · `[REFACTOR]` · `[DOCS]` · `[CONSTRAINT]` · `[HARNESS]` · `[PLAN]`. Convention — not yet mechanically enforced.

## Context Management

- 대형 작업 시작 시 `handoff-{feature}.md` 작성 (컨텍스트가 신선할 때).
- 컨텍스트가 채워지면 인플레이스 압축이 아닌 **리셋 + 핸드오프 파일** 전략 사용.
- 징후: 앞 기능은 완전히 구현하고 뒤는 스텁만 채움 → 즉시 리셋.

## Working Notes

- **KIS 토큰 발급은 분당 1회 제한** — live auth 검증 루프 반복 금지.
- `KIS_ENV`: `real` / `demo` / `vps` / `paper`. 잘못된 값은 조용히 production URL로 라우팅됨.
- HTMX 라우트는 `HX-Request` 헤더로 partial/full 응답 분기. 일관성 유지.
- Tailwind CSS는 standalone CLI (`bin/tailwindcss`). Node.js 불필요. `make css-watch`로 개발.

## Language Policy

- 코드, 커밋, docs/: English
- 사용자 커뮤니케이션: Korean
