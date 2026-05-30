# Backlog

## Python → Go rewrite (Phase 9–10)

Branch `feat/go-rewrite`. Phases 0–8 done. Companion: `handoff-go-rewrite.md`.

### Phase 9 — Insights / LLM

Python oracle: `web/routes/insights.py`, `services/portfolio_insight_service.py`,
`services/llm/ollama_client.py`, `services/llm/prompt_templates.py`, templates
`insights/view.html`, `_narrative.html`, `_rebalance_xai.html`, `_qa.html`.

- [ ] `OllamaClient.chat` — POST /api/chat, tools, format json, `OllamaUnavailableError`
- [ ] prompt templates (Korean): narrative / rebalance-xai / qa / qa-json-fallback + 4 tool schemas
- [ ] `PortfolioInsightService` — `generate_narrative(daily/weekly)`, `explain_rebalance`, `answer_question` (4 tools: group_summary/top_movers/holding_value/deposit_history, 3-iter cap, JSON fallback)
- [ ] `GET /insights`, `GET /insights/narrative?period=`, `GET /insights/rebalance-xai`, `POST /insights/qa`
- [ ] templ view + 3 partials; register; parity (mock Ollama)
- [ ] env: `OLLAMA_MODEL`/`HOST`/`TIMEOUT_SEC`/`NUM_CTX`

### Phase 10 — Cutover

- [ ] full `ServiceContainer` env wiring complete + `close()` lifecycle
- [ ] move `src/.../web/static` + tailwind input → `internal/web/static`; update Makefile css paths + `staticDir()`
- [ ] Dockerfile → multi-stage Go (`CGO_ENABLED=0`, scratch/alpine); docker-compose → Go binary
- [ ] pre-commit → golangci-lint + `go test` + `templ generate --check` + `sqlc diff` (drop ruff/pyright/bandit/pytest)
- [ ] CI (`.github/workflows/ci.yml`) → Go-only; coverage gate 85% (exclude generated: `internal/db/sqlc`, `*_templ.go`, `cmd`)
- [ ] rewrite `docs/*.md` + `AGENTS.md` golden principles for Go toolchain
- [ ] `.claude/settings.local.json` permissions: `uv run`/pytest → go/golangci
- [ ] DELETE `src/`, `tests/` (py), `pyproject.toml`, `uv.lock`, `.venv`, `.bandit`, `.pre-commit-config.yaml` (py hooks)
- [ ] final 38-route parity sweep + `docker compose up` smoke
- [ ] remove `scripts/parity_check.sh` + `handoff-go-rewrite.md` + `TASKS-go-rewrite.md` (oracle gone)

---

- [x] [harness] Define dormant `tasks.md` schema/status for no-active-sprint sessions

## Review follow-ups from PR #65

- [x] [docs] `runbook.md` — KST 마이그레이션 이후 기존 DB 행은 `+00:00` offset 으로 남아 있음을 명시 (다음 업데이트 시 자연 복구).
- [x] [harness] `.data/kis_sync.log` JSONL 로테이션 — 크기 기반(`_MAX_SYNC_LOG_BYTES = 10 MB`), `.log.1` 백업으로 rotate.
- [x] [feat] Q&A soft wall-clock deadline 도입 — `_QA_DEADLINE_SEC = 120.0`, `time.monotonic()` 기반, primary/fallback 루프 + no-tools call 모두 가드.

## 종목명 표시 개선 — Wave 2 (feat/stock-name-wave1 머지 후)

- [x] [feat] Phase 4: KIS 잔고 응답(`output1.prdt_name`) 파싱 → `KisHoldingPosition.name` 신설 → `KisAccountSyncService`가 신규/기존 stock에 이름 저장.
- [x] [feat] Phase 5: 해외 종목 이름 조달 — `parse_us_price` 다중 필드명 폴백은 완료. 잔여: `KisOverseasInfoClient` 신설(tr_id는 KIS 공식 문서 재확인 필수) + `KisUnifiedPriceClient` 해외 분기에 info 폴백 추가. (#79; KisUnifiedPriceClient overseas fallback wired in `container.py:278`)

## Local Ollama AI 인사이트 — Stage 1 잔여 + Stage 2 (branch `feat/llm-insights-stage1` 머지 후)

### Stage 1 잔여 — 수동 검증 필요

- [ ] [verify] 로컬에서 `ollama serve` + `ollama pull 0xIbra/supergemma4-26b-uncensored-gguf-v2:Q4_K_M` 후 `.env` 에 `OLLAMA_MODEL` 설정 → `make dev` 로 `/insights` 탭 3종 실제 응답 품질 확인. 일/주간 토글, 리밸런싱 XAI JSON 파싱, Q&A 도구 호출(4종) 전부 브라우저에서 클릭 검증.
- [ ] [verify] `supergemma4-26b` tool-calling 실패 시 `format=json` fallback 경로 실제 동작 확인. 필요 시 `prompt_templates.QA_JSON_FALLBACK_SYSTEM_PROMPT` 튜닝.
- [ ] [feat] 필요 시 내러티브 스트리밍 (`httpx.AsyncClient` + async 라우트) 도입 — 현재는 non-stream + 스피너. 26B 응답 지연이 UX 를 해치면 진행.

### Stage 2 — KIS 수급 + B2 이상치 감지

- [ ] [feat] `services/kis/kis_domestic_investor_client.py` — `/uapi/domestic-stock/v1/quotations/inquire-investor` 래퍼. tr_id 는 KIS 공식 문서 재확인 필수. 기존 `KisBaseClient` 상속 + `TokenManager` 공유.
- [x] [feat] `services/database.py` + `repositories/investor_flow_repository.py` — 외인/기관/개인 일별 순매수 캐시 테이블(ticker, flow_date, foreign/institution/individual net qty/krw) + `KisDomesticInvestorClient` container wiring + migration smoke test. (PR TBD)
- [ ] [feat] `services/flow_anomaly_service.py` — z-score > 2 또는 5일 연속 순매수 감지 → `AnomalySignal` 반환. 보유 종목 대상 lazy 조회(당일 캐시 우선).
- [ ] [feat] `PortfolioInsightService.explain_anomalies()` + 라우트 `/insights/alerts` (HTMX partial) + `view.html` 네 번째 탭 또는 배너.
- [ ] [docs] `runbook.md` — KIS demo 환경에서 수급 데이터 제한 가능성 명시 + `pytest -m integration` 커버.

### Stage 1 주의사항 (Stage 2 착수 전 확인)

- Ollama env 미설정 시 `/insights` 는 "service unavailable" 배너로 graceful fallback — Stage 2 에서도 동일 패턴 유지.
- 숫자 = Python / 산문 = LLM 원칙: Stage 2 의 anomaly rationale 도 신호 수치는 Python 이 렌더하고 LLM 은 "왜 주목해야 하나" 문장만 생성.