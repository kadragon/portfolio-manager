# plan.md

## 웹 인터페이스 — FastAPI + HTMX/Jinja2

> CLI 대신 브라우저에서 포트폴리오 조회/수정/리밸런싱을 수행할 수 있도록 로컬 웹 서버를 추가한다.
> 기존 서비스/레포지토리 계층을 그대로 재사용하고, 프레젠테이션 레이어만 새로 구현한다.
> 상세 설계: `.claude/plans/bright-puzzling-bonbon.md` 참조.

### Phase 1: 스켈레톤


### Phase 2: Groups CRUD


### Phase 3: Accounts + Holdings CRUD


### Phase 4: Deposits CRUD


### Phase 5: Rebalance


### Phase 6: 마무리


## Review Backlog

### PR #54 — Migrate data layer from Supabase to SQLite + Peewee ORM (2026-03-28)

- [x] `migrate_supabase_to_sqlite.py` — 일회성 스크립트 삭제 (Supabase 의존 제거 완료)
- [x] `init_db()` — `_default_db_path()`로 절대경로 해석 + `PORTFOLIO_DB_PATH` 환경변수 지원
- [x] `updated_at` — `BaseModel.save()` 오버라이드로 자동 갱신
- [x] `OrderExecutionModel` — append-only 모델이므로 `updated_at` 불필요. 리스크 수용

## Security Fixes — portfolio-manager

> Fix all open GitHub security alerts for this repository.

### Dependabot Alerts

- [ ] Pygments ReDoS vulnerability (LOW, GHSA-5239-wwwm-4pmq) — no patched version available yet; current version 2.19.2. Monitor for a fix release and upgrade when available

### Code Scanning Alerts

- [x] Fix actions/missing-workflow-permissions: Workflow does not contain permissions (WARNING) — .github/workflows/ci.yml:11-36. Add explicit `permissions:` block (e.g., `contents: read`) to the workflow or job
