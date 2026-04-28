# Runbook

## Quick Start

### Prerequisites

- Python 3.13+ (`python --version`)
- [uv](https://docs.astral.sh/uv/) (`uv --version`)
- `.env` file in project root (copy from `.env.example` if it exists, or see Environment Variables below)

### Setup

```bash
uv sync                    # install Python dependencies
make setup                 # download Tailwind standalone CLI + DaisyUI bundle
```

### Run

```bash
make dev                   # web server (port 8000) + CSS watcher
# or separately:
make css-watch &           # CSS auto-rebuild on template changes
uv run portfolio-web       # http://127.0.0.1:8000
```

### Docker (alternative)

See `docs/docker.md` for full Docker workflow. Quick start:

```bash
docker compose up -d --build web
# http://localhost:8000
```

## Build

| Command | Purpose |
|---------|---------|
| `make css-build` | Production CSS (minified) |
| `make css-watch` | Dev CSS watcher |
| `uv run portfolio-web` | Start web server |
| `uv run pyright` | Type check |
| `uv run ruff check` | Lint |
| `uv run ruff format` | Format |

## Test

| Command | Purpose |
|---------|---------|
| `uv run pytest` | Unit tests (excludes integration, coverage ≥85% required) |
| `uv run pytest -m integration` | Integration tests (hits real KIS API) |
| `uv run pytest tests/web/` | Web route tests only |
| `uv run pytest tests/repositories/` | Repository tests only |
| `uv run pytest -k "test_name"` | Single test by name |
| `uv run pytest --cov-report=html` | HTML coverage report → `htmlcov/` |

**Coverage threshold: 85% branch coverage.** CI fails below this.

Integration tests require KIS credentials in `.env`. Do not run in CI.

## Lint & Format

| Command | Purpose |
|---------|---------|
| `uv run ruff check` | Lint check |
| `uv run ruff check --fix` | Lint with auto-fix |
| `uv run ruff format` | Format all Python files |
| `uv run ruff format --check` | Check formatting without writing |
| `uv run bandit -r src scripts` | Security scan |
| `uv run pyright` | Type check |

Pre-commit runs all of the above automatically on staged files. Install with:

```bash
uv run pre-commit install
```

## Sweep

Harness garbage collection. Runs ruff + pyright + arch tests + bandit, plus harness-freshness checks (AGENTS.md references, CLAUDE.md pointer, AGENTS.md size).

| Command | Purpose |
|---------|---------|
| `bash scripts/sweep.sh` | Full sweep |
| `bash scripts/sweep.sh --quick` | Lint only |

**Trigger policy:** manual. Run between features or before opening a PR. Findings are appended to `backlog.md` as checkbox items.

## Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `KIS_APP_KEY` | No* | KIS API app key | `PSxxxxxxxx` |
| `KIS_APP_SECRET` | No* | KIS API app secret | `...` |
| `KIS_ENV` | No | `real` / `demo` / `vps` / `paper` | `real` |
| `KIS_CUST_TYPE` | No | Customer type: `P` (individual) or `B` (corporate) | `P` |
| `KIS_CANO` | No | KIS account number (8 digits) | `12345678` |
| `KIS_ACNT_PRDT_CD` | No | Account product code | `01` |
| `KIS_ACCOUNT_NO` | No | Alternative: full 10-digit account number | `1234567801` |
| `KIS_APP_KEY_2` | No | Second KIS app key (for multi-account) | |
| `KIS_APP_SECRET_2` | No | Second KIS app secret | |
| `KIS_PRDT_TYPE_CD` | No | Product type code for stock info queries | `300` |
| `KIS_DOMESTIC_INFO_TR_ID` | No | TR ID for domestic stock info | `CTPF1002R` |
| `USD_KRW_RATE` | No† | Fixed USD/KRW rate (overrides EXIM) | `1350.00` |
| `EXIM_AUTH_KEY` | No† | EXIM API key for live exchange rates | |
| `OLLAMA_HOST` | No | Ollama server URL (default `http://localhost:11434`) | `http://localhost:11434` |
| `OLLAMA_MODEL` | No‡ | Ollama model tag for AI insights. Unset → `/insights` page disabled | `0xIbra/supergemma4-26b-uncensored-gguf-v2:Q4_K_M` |
| `OLLAMA_TIMEOUT_SEC` | No | Per-request timeout in seconds (default `60`) | `120` |
| `OLLAMA_NUM_CTX` | No | Override context window size | `8192` |

\* App runs without KIS credentials — price fetching and account sync are disabled.  
† At least one of `USD_KRW_RATE` or `EXIM_AUTH_KEY` is needed for overseas stock valuation.  
‡ `OLLAMA_MODEL` unset ⇒ `/insights` page renders a "service unavailable" banner. A running `ollama serve` is required when the variable is set; app startup does not block on the Ollama server.

KIS token files are cached in `.data/kis_token_{key_id}.json`. Mount `.data/` in Docker to persist tokens across restarts (avoids rate-limit violations on restart).

## Common Failures

### KIS token issuance fails on startup

**Symptom:** `Could not initialize price service: ...` in startup logs.  
**Cause:** Invalid credentials, wrong `KIS_ENV`, or rate limit hit (1 token/min).  
**Fix:** Check `KIS_APP_KEY` / `KIS_APP_SECRET` / `KIS_ENV`. Wait 60 seconds if rate-limited. Token is cached in `.data/` — if the file is stale or corrupted, delete it.

### CSS not updating in browser

**Symptom:** Template changes don't reflect in browser.  
**Cause:** `make css-watch` not running, or `bin/tailwindcss` not downloaded.  
**Fix:** Run `make setup` first, then `make css-watch`.

### Coverage below 85%

**Symptom:** `FAIL Required test coverage of 85% not reached.`  
**Cause:** New code added without tests.  
**Fix:** Write tests for the uncovered lines shown in the coverage report. Run `uv run pytest --cov-report=term-missing` to see exact lines.

### pre-commit hook fails on pyright

**Symptom:** `error: Cannot find implementation or declaration file for module '...'`  
**Cause:** New dependency not installed, or `.venv` out of sync.  
**Fix:** Run `uv sync` to reinstall, then re-commit.

### Docker: `.data` permission error on Linux

**Symptom:** `PermissionError: [Errno 13] Permission denied: '.data/kis_token_1.json'`  
**Cause:** Container created files as root, host user can't write.  
**Fix:** `sudo chown -R "$(id -u):$(id -g)" .data && LOCAL_UID=$(id -u) LOCAL_GID=$(id -g) docker compose up -d --build web`

### `/insights` shows "AI 인사이트 서비스가 설정되지 않았습니다"

**Symptom:** The AI 인사이트 tab banners a setup warning.  
**Cause:** `OLLAMA_MODEL` is unset, price service failed to start (no KIS key), or the Ollama server is unreachable.  
**Fix:** Start `ollama serve`, pull the desired model (`ollama pull <tag>`), set `OLLAMA_MODEL` in `.env`, and restart. Network failures while the page is open surface as warning banners under each tab — the page stays functional with Python-computed numbers.

### In-memory DB not reset between tests

**Symptom:** Tests pass individually but fail when run together (state leaks).  
**Cause:** Test fixture not scoped correctly, or shared container instance.  
**Fix:** Use `function`-scoped `test_container` fixture. Check `tests/conftest.py`.

### Timezone / KST migration

**Background:** Timestamps were migrated from UTC (`+00:00`) to KST (`+09:00`) in a prior release.  
**Effect:** Pre-migration rows retain their original `+00:00` `created_at` permanently — this column is immutable after insert. `updated_at` columns update to KST offset on the next write to that row. No manual data migration is needed; affected rows self-heal naturally over time.

**Affected models** (`created_at` + `updated_at`, both self-heal on next write):
`GroupModel`, `StockModel`, `AccountModel`, `HoldingModel`, `DepositModel`, `StockPriceModel`
— all defined in `src/portfolio_manager/services/database.py`.

**Self-heal mechanism:** `BaseModel.save()` (`database.py:29-32`) sets `updated_at = now_kst()` on every write. Pre-migration rows self-heal on their next update; no batch job is needed.

**Exception — `OrderExecutionModel`:** insert-only, no `updated_at` column. Its `created_at` rows written before migration remain at `+00:00` permanently and will not self-heal.
