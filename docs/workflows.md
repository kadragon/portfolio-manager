# Workflows

Six workflows. Pick one primary per cycle. Side-effects permitted per the table at the bottom.

## `plan` — Spec Generation

Expand a short prompt into a full implementation spec.

1. Write `docs/design/{feature}.md`: user stories, tech design, phased feature list. **No granular implementation details** — errors cascade downstream.
2. Review with user. Do not proceed until approved.
3. Generate `backlog.md` items from the approved spec.

Skip for trivial changes.

## `code` — Implementation

The primary cycle for behavioral changes.

**Step 1: Scope check**
Before writing code, if the change touches ≥3 directories or `services/kis/`: launch an Explore agent to map the area first.

**Step 2: Define done**
Write testable acceptance criteria before touching production code. What does "complete" look like in concrete, verifiable terms?

**Step 3: Implement**
- TDD: failing test → passing → refactor.
- Keep coverage ≥85% (branch). CI enforces this.
- No layer boundary violations (GP-1, GP-3). `pytest tests/arch/` enforces this.
- Run `uv run ruff check` + `uv run pyright` before committing.

**Step 4: Verify**
Pre-commit hooks run pyright, ruff, bandit, and pytest automatically. All must pass.

## `draft` — Documentation

Write or update `docs/`. Ground every claim in current code. Do not modify production code. If the doc reveals a missing constraint, add to `tasks.md`.

## `constrain` — Architectural Enforcement

1. Write the structural test or lint rule **first**.
2. Run it.
3. If existing code violates → add remediation to `backlog.md`. Do not fix here.
4. Update `docs/architecture.md`.

## `sweep` — Garbage Collection

Run between features or on a schedule.

- Scan for stale comments, unused imports, outdated docs.
- Record findings in `tasks.md` tagged as `[doc]` / `[constraint]` / `[debt]` / `[harness]`.
- Fix trivials inline.
- Harness simplification check: "Is this component still compensating for a real model limitation? Can it be removed?"

## `explore` — Research

State the question → research/prototype → report options and trade-offs → **no commit**. If approved, flows into `plan` or `code`.

---

## Permitted Side-Effects

| Primary workflow | Permitted side-effect |
|------------------|-----------------------|
| `code` | Add `[doc]` or `[constraint]` item to `tasks.md` when discovering issues |
| `code` | Update relevant `docs/` after implementation |
| `draft` | Add `backlog.md` item when doc reveals missing behavior |
| `sweep` | Fix trivial `[doc]` items inline |

Writing production code during `draft` or `sweep` is not permitted.

---

## Context Management

When context fills during a long task:

1. **Reset over compaction.** In-place compaction doesn't resolve context anxiety — a full reset with a handoff file is safer.
2. **Write `handoff-{feature}.md` at session start.** Create it when context is fresh and the plan is clear. Delete when the feature is complete.
3. **Anxiety symptoms:** first 3 features fully implemented, later ones stubbed out; sudden "the rest can be done similarly." If this appears, reset immediately.
