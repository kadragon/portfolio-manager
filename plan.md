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

- [ ] `migrate_supabase_to_sqlite.py` — Supabase API 페이지 크기 초과 시 데이터 잘림 가능. 페이지네이션 추가 필요 (source: Codex) — scripts/migrate_supabase_to_sqlite.py:77
- [ ] `init_db()` 기본 경로가 상대경로(`.data/portfolio.db`). CWD에 따라 다른 위치에 DB 생성될 수 있음. 절대경로 또는 환경변수 기반으로 변경 (source: Claude) — services/database.py:153
- [ ] `updated_at` 수동 관리 패턴에 누락 위험. `BaseModel.save()` 오버라이드로 자동 갱신 검토 (source: Gemini)
- [ ] `OrderExecutionModel`에 `updated_at` 필드 없음 (다른 모델과 비일관). append-only라 현재는 불필요하나 향후 검토 (source: Claude, Gemini)
