<!--
Schema / lifecycle:
  Active sprint  — open [ ] items grouped by PR or feature branch below this header.
  Dormant        — no active sprint; file starts with "# (dormant)" and retains only open [debt/blocked] items.
-->

# (dormant) Tasks — Deferred from PR reviews

## Review Backlog

### PR #100 — [HARNESS] Upgrade harness to Level 2 (2026-05-27)

- [ ] [doc] `docs/delegation.md` Model Selection table — pin model versions or add note to resolve from `~/.claude/settings.json`; bare tier names (sonnet/opus) may drift on model upgrade (source: pr-review-toolkit:review-pr) — `docs/delegation.md:52`
- [ ] [doc] `docs/adr/README.md` ADR template missing `## Options Considered` section — AGENTS.md says ADRs document "options considered" but template has only Context/Decision/Consequences (source: pr-review-toolkit:review-pr) — `docs/adr/README.md`
- [ ] [doc] `docs/eval-criteria.md` Correctness criterion "How to test: Run the feature manually" — replace with specific `go test` command or scenario list for CI-reproducibility (note: original item said "pytest command"; repo is now Go) (source: review) — `docs/eval-criteria.md:37`

### PR #111 — [FIX] remove KIS key ID value from fallback log (2026-06-02)

- [ ] [constraint] `resolveSyncService` has no unit tests — add table-driven test covering: nil keyID, found key, key=1 not found (no log), key≠1 not found (warning log path) (source: pr-review-toolkit:review-pr) — `internal/container/container.go:225`
