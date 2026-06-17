# Delegation

## When to Delegate

Delegate when work is **separable** — self-contained task with clear I/O and a verifiable exit criterion.

### Hard Gates (mandatory — do not bypass)

| Trigger | Action |
|---------|--------|
| Same failure 2× | `advisor` tool → if unresolved: `codex:rescue` |
| Task has 2+ valid interpretations | Grill protocol (one question at a time) before any code |
| External blocker (auth, credentials, network) | Pause and report to user |

### Background Gates (non-blocking — prefer delegation)

| Trigger | Suggested target |
|---------|-----------------|
| Deep root-cause investigation | `codex:rescue` |
| Second implementation pass | `codex:rescue` |
| Multi-file refactor with clear scope | `codex:rescue` |
| Search across many files | `Explore` agent (or `codex:rescue` if unavailable) |

## Spawn Prompt Contract

Every delegation brief must include all four fields:

```
Goal: [one sentence — what must be true when done]
Constraints: [what must NOT change; layer rules; style requirements]
Exit criterion: [single verifiable test — "pytest X passes" or "command exits 0"]
Files / commands needed: [list relevant paths and run commands]
```

Missing any field → delegation is vague → likely to fail → do not delegate until complete.

## Pattern Selection

```
Is the task separable with clear I/O?
├── No  → inline (do it directly)
└── Yes → Is it stuck or repeating?
          ├── Yes → codex:rescue
          └── No  → Is it a broad search across many files?
                    ├── Yes → Explore agent
                    └── No  → codex:rescue with spawn contract
```

## Model Selection

| Role | Model | Use for |
|------|-------|---------|
| Implementation | sonnet | Feature code, bug fixes |
| Investigation | sonnet | Root cause, search |
| Architecture | opus | Design decisions, trade-off analysis |
| Structural grading | haiku | Lint, format checks |

> Tier names (sonnet/opus/haiku) are aliases resolved by the runtime — pin exact model IDs in `~/.claude/settings.json` if version stability matters.

## Anti-Patterns

- **Delegating ambiguous tasks** — write the spawn contract first; if you can't complete it, the task isn't ready to delegate.
- **Delegating context-dependent work** — if the subagent needs your current session context to proceed, keep it inline.
- **Delegating trivial edits** — single-function rewrites or typo fixes are faster inline.
