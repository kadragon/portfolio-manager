# plan.md

## 웹 인터페이스 — FastAPI + HTMX/Jinja2

> CLI 대신 브라우저에서 포트폴리오 조회/수정/리밸런싱을 수행할 수 있도록 로컬 웹 서버를 추가한다.
> 기존 서비스/레포지토리 계층을 그대로 재사용하고, 프레젠테이션 레이어만 새로 구현한다.
> 상세 설계: `.claude/plans/bright-puzzling-bonbon.md` 참조.

### Phase 1: 스켈레톤

- [x] pyproject.toml에 fastapi, uvicorn, jinja2, python-multipart 의존성 추가 + portfolio-web 엔트리포인트
- [x] web/app.py — app factory, lifespan(container 생성/종료), Jinja2 템플릿/필터 설정
- [x] web/deps.py — get_container, get_templates 의존성 헬퍼
- [x] templates/base.html — Pico CSS + HTMX + 네비게이션 레이아웃
- [x] routes/dashboard.py — GET / 대시보드 (보유현황 테이블 + 그룹 요약 + 투자 요약)
- [x] 동작 확인: portfolio-web 실행 → http://127.0.0.1:8000 접속

### Phase 2: Groups CRUD

- [x] routes/groups.py — 그룹 목록 조회 (GET /groups) + list.html
- [x] 그룹 생성 (POST /groups) + _form.html, _row.html
- [x] 그룹 수정 (GET /groups/{id}/edit, PUT /groups/{id})
- [x] 그룹 삭제 (DELETE /groups/{id})
- [x] 그룹 내 종목 관리 (GET/POST/DELETE /groups/{id}/stocks) + stocks.html

### Phase 3: Accounts + Holdings CRUD

- [x] routes/accounts.py — 계좌 목록/생성/수정/삭제 + 템플릿
- [x] 계좌 내 보유 관리 (GET/POST/PUT/DELETE holdings) + 템플릿
- [x] KIS 계좌 동기화 (POST /accounts/{id}/sync)

### Phase 4: Deposits CRUD

- [x] routes/deposits.py — 입금 목록/생성/수정/삭제 (중복 날짜 처리 포함) + 템플릿

### Phase 5: Rebalance

- [x] routes/rebalance.py — 리밸런싱 추천 조회 (GET /rebalance) + view.html
- [x] 주문 실행 (POST /rebalance/execute) + _result.html

### Phase 6: 마무리

- [x] 대시보드 자동 새로고침 (HTMX hx-trigger)
- [x] 플래시 메시지 (성공/에러 피드백)
- [x] 네비게이션 active 상태 표시
