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


## Security Fixes — portfolio-manager

> Fix all open GitHub security alerts for this repository.

### Dependabot Alerts

- [ ] Pygments ReDoS vulnerability (LOW, GHSA-5239-wwwm-4pmq) — no patched version available yet; current version 2.19.2. Monitor for a fix release and upgrade when available
