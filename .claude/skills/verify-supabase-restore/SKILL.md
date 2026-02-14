---
name: verify-supabase-restore
description: Verify Supabase project status/restore flow and CLI startup behavior after Supabase connectivity changes.
---

# Verify Supabase Restore

## Purpose
1. Ensure Supabase Management API status parsing and restore request handling are correct.
2. Validate project-ready polling behavior does not fail too early during transient states.
3. Confirm CLI startup gate behavior is resilient for ACTIVE/RESTORING/PAUSED states.
4. Keep focused tests aligned with restore semantics and startup expectations.

## When to Run
- Changes in `src/portfolio_manager/services/supabase_project.py`.
- Changes in `src/portfolio_manager/cli/main.py` around startup flow.
- Changes in Supabase auto-resume behavior or env var handling.
- Test updates related to paused/restore startup behavior.

## Related Files

| File | Purpose |
| --- | --- |
| `src/portfolio_manager/services/supabase_project.py` | Project status checks, restore request, and readiness polling |
| `src/portfolio_manager/cli/main.py` | CLI startup gate using restore status |
| `src/portfolio_manager/services/supabase_client.py` | Existing Supabase auto-resume behavior baseline |
| `tests/services/test_supabase_project.py` | Focused tests for status/restore/wait behavior |
| `tests/ui/test_rich_main.py` | CLI startup behavior tests |
| `tests/test_supabase_auto_resume.py` | Existing auto-resume contract tests |

## Workflow

### Step 1: Validate status and restore semantics
**Tool:** Read, Grep

- Confirm `ProjectStatus` covers `ACTIVE_HEALTHY`, `INACTIVE_PAUSED`, `RESTORING`, and `UNKNOWN`.
- Confirm `restore_project` treats accepted restore responses as success.

```bash
grep -n "class ProjectStatus\\|ACTIVE_HEALTHY\\|INACTIVE_PAUSED\\|RESTORING\\|restore_project\\|status_code in" src/portfolio_manager/services/supabase_project.py
```

**PASS:** Status mapping is explicit and restore success accepts Management API success codes.
**FAIL:** Restore only accepts one code that breaks known API behavior, or status mapping is incomplete.

### Step 2: Validate readiness polling behavior
**Tool:** Read, Grep

- Confirm `wait_for_project_ready` keeps polling until ACTIVE or timeout.
- Ensure transient non-active states do not cause immediate false failures.

```bash
grep -n "def wait_for_project_ready\\|while elapsed < max_wait_seconds\\|ProjectStatus.ACTIVE\\|return False" src/portfolio_manager/services/supabase_project.py
```

**PASS:** Function returns `True` only on ACTIVE and otherwise waits until timeout.
**FAIL:** Early failure path exists for transient paused/restoring states after restore request.

### Step 3: Validate CLI startup gate behavior
**Tool:** Read, Grep

- Confirm `_ensure_supabase_ready` blocks only when startup truly cannot proceed.
- Confirm RESTORING state continues startup with warning and PAUSED state aborts with actionable message.

```bash
grep -n "def _ensure_supabase_ready\\|ProjectStatus.ACTIVE\\|ProjectStatus.PAUSED\\|ProjectStatus.RESTORING\\|return True\\|return False" src/portfolio_manager/cli/main.py
```

**PASS:** ACTIVE and RESTORING continue startup; PAUSED blocks startup.
**FAIL:** RESTORING hard-aborts startup or PAUSED silently proceeds.

### Step 4: Run targeted tests
**Tool:** Bash

```bash
uv run pytest -q \
  tests/services/test_supabase_project.py \
  tests/ui/test_rich_main.py \
  tests/test_supabase_auto_resume.py
```

**PASS:** All tests pass.
**FAIL:** Any failing test indicates restore/startup regression; fix behavior before merge.

## Output Format

| Check | Status | Notes |
| --- | --- | --- |
| Status/restore semantics | PASS/FAIL |  |
| Polling behavior | PASS/FAIL |  |
| CLI startup gate | PASS/FAIL |  |
| Tests | PASS/FAIL |  |

## Exceptions

1. Missing `SUPABASE_ACCESS_TOKEN` can intentionally disable auto-restore and is not a violation by itself.
2. Integration-only network instability should not be treated as logic regression when focused unit tests pass.
3. Additional informational console messages are acceptable if startup decision logic remains unchanged.
