# Handoff — Python → Go rewrite

Branch: `feat/go-rewrite`. Full plan: `~/.claude/plans/witty-wobbling-lemur.md`.
Strategy: **clean replace** (Go replaces `src/` at cutover). Python kept on disk
**only** as a dev-time parity oracle until each slice matches; deleted in Phase 10.

## Status

| Phase | State |
|-------|-------|
| 0 — scaffold + DB-compat types | ✅ |
| 1 — groups slice (full stack) | ✅ |
| 2 — stocks | ✅ |
| 3 — accounts | ✅ |
| 4 — holdings | ✅ |
| 5 — deposits | ✅ |
| 6 — portfolio/price/exchange/dashboard | ✅ |
| 7 — rebalance | ✅ (`b7a73e4`) |
| 8 — KIS account sync | ✅ (`a9f7826`) |
| 9 — insights/LLM | ❌ dropped |
| 10 — cutover | ⬜ see `backlog.md` |

## Stack (locked)

Echo v4 · templ · sqlc · modernc.org/sqlite (pure Go, CGO_ENABLED=0) ·
shopspring/decimal · google/uuid. Module `github.com/kadragon/portfolio-manager`.
Tools not in PATH — call `/Users/kadragon/go/bin/{sqlc,templ}` or `make go-gen`.

## Layout established

```
cmd/portfolio-web/main.go      entrypoint (Echo, graceful shutdown, /static reuse)
internal/db/schema.sql         DDL (production parity); db.go Open/OpenMemory
internal/db/sqlc/              generated (committed)
internal/{uuidx,numeric,datex,ktime}/   Peewee-compatible SQL types
internal/models/               domain structs
internal/repositories/         GP-1 owns DB; wraps sqlc
internal/container/            composition root (New / NewWithQueries)
internal/web/handlers/         Echo handlers; render.go sets status before body
internal/web/templates/        *.templ + helpers.go (navItems, raw-HTML helpers)
internal/web/format/           Jinja-filter equivalents
internal/arch/arch_test.go     GP-1/GP-3 import boundaries (skips _test.go)
query/queries.sql              sqlc source
scripts/parity_check.sh        dev-time HTML parity harness
```

## Critical parity facts (verified against real .data/portfolio.db)

- **UUID** stored as 32-char hex, no dashes (`uuidx` handles). HTML renders the
  **dashed** form (`g.ID.String()`) — matches Python `str(uuid)`.
- **Decimal**: NUMERIC affinity → whole values come back `int64`, fractional
  `float64`. `numeric.Decimal.Value()` writes int64 when whole (keeps existing
  rows integer), string otherwise. Fractional rows already exist.
- **Datetime** TEXT `2026-01-03 13:21:44.873677+00:00` (existing rows +00:00;
  new rows +09:00 KST). **Date** TEXT `YYYY-MM-DD`. Timestamps set in Go code
  (`ktime.Now()`), never SQL defaults.
- **Schema column order** must match on-disk: migrated cols come LAST
  (`stocks.name`; `accounts.kis_account_no, kis_api_key_id`) because sqlc scans
  `SELECT *` in schema order. Index names = Peewee's (`stockmodel_group_id`,
  `depositmodel_deposit_date`, …) so `CREATE INDEX IF NOT EXISTS` no-ops on prod.
- **List order**: Peewee `.select()` has no ORDER BY → emit rowid/insertion
  order. Do NOT add `ORDER BY created_at` (breaks parity).
- **No SQL DEFAULT clauses** — Peewee applies defaults in Python; app supplies
  every column on insert.

## Parity verification (the loop)

`bash scripts/parity_check.sh` boots Python (uvicorn :8001) + Go (:8079) on
identical DB copies, normalizes HTML (tag-boundary whitespace, doctype case,
`&#39;`/`&#34;`/`&amp;` entity equivalence), diffs. Extend the route list per
slice. Treat MATCH as the per-route exit criterion. It cleans up its own
servers. `PARITY_SRC_DB` overrides the source DB.

## Conventions / gotchas learned (IMPORTANT)

- **One shell action per tool call when a non-zero exit is possible** — the
  harness cancels every other call in the same parallel batch when one errors.
  Batch only Reads/Writes that can't fail. Several lost edits traced to this.
- **A pre-commit hook runs on every commit** (pyright/ruff/bandit/pytest, all
  skip-or-pass for Go-only changes). Its stash/restore flips
  `.github/workflows/ci.yml`'s file mode (755↔644), leaving it perpetually
  "modified". Never stage ci.yml; `git checkout -- .github/workflows/ci.yml`
  after committing. It is unrelated to this work.
- Shell output sometimes appears duplicated and `git log` can show a stale HEAD
  for a beat — verify commit state by writing `git log --format=... > file` and
  Reading the file, not by eyeballing inline output.
- zsh: don't `echo $VAR` a value that may start with hex digits (UUID) — math
  expansion error. Use files / printf.
- templ can't express a `<form>` whose owner spans multiple `<td>` — emit such
  partials via `templ.Raw(...)` with `html.EscapeString` on dynamic values
  (see `internal/web/templates/helpers.go: groupFormHTML`).
- FastAPI Form parity: required field absent → 422; optional with default →
  default when absent but 422 when present-and-unparseable; bad path UUID → 422.
- golangci `db.go` MkdirAll G703 is excluded (operator-controlled path).

## Per-slice recipe (repeat)

1. Read the Python route/repo/model/templates for the slice.
2. Add sqlc queries → `make go-gen`; check generated types in `internal/db/sqlc`.
3. Repository (+ in-memory test), domain model, container wiring.
4. templ templates (+ format helpers) — copy Jinja structure exactly.
5. Echo handlers (+ httptest test); register in `cmd/portfolio-web/main.go`.
6. `make go-build go-vet go-lint go-test` green; arch test green.
7. Extend `scripts/parity_check.sh` with the slice's routes → all MATCH.
8. Commit `[FEAT] Phase N: <slice>` (then restore ci.yml mode).

## Deferred to cutover (Phase 10)

- Coverage gate: currently ~26% (generated code + cmd/container dilute it). Add
  generated-file exclusions and raise to 85% then.
- Dockerfile (multi-stage, CGO_ENABLED=0), docker-compose, pre-commit, CI flip,
  docs/AGENTS.md rewrite, delete `src/` + Python tooling.
  `.claude/settings.local.json` permissions still reference `uv run`/pytest.
