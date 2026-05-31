# Backlog

## Python → Go rewrite (Phase 10)

Branch `feat/go-rewrite`. Phases 0–8 done; Phase 9 (LLM/insights) dropped. Companion: `handoff-go-rewrite.md`.

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
