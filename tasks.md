<!--
Schema / lifecycle:
  Active sprint  — open [ ] items grouped by PR or feature branch below this header.
  Dormant        — no active sprint; file starts with "# (dormant)" and retains only open [debt/blocked] items.
-->

# Tasks — Deferred from PR reviews

## Review Backlog

### PR #151 — [FIX] validate -security-group against KIS allowlist in pm stock update (2026-07-14)

- [ ] [debt] `ValidSecurityGroup` allowlist may reject unknown-but-legitimate future KIS `scty_grp_id_cd` codes not in the initial 8; this PR restores CLI-side enforcement of the existing allowlist but does not extend the allowlist itself, so the underlying concern from PR #137 is still unaddressed — extend the allowlist if KIS adds new codes (source: review, contest-round) — `internal/models/stock.go:ValidSecurityGroup` *(deferred: no concrete new KIS code to add)*

### PR #157 — [REFACTOR] migrate rebalance snapshot.py to pm snapshot subcommand (2026-07-16)

- [ ] [debt] `pm snapshot` boots the shared `container.New("")`, which opens the DB read-write and applies pending schema migrations before dispatch — the removed `snapshot.py` opened SQLite `mode=ro`, so on a missing/pre-migration DB the new command can create/migrate production state where the old one could not. This is a pre-existing property of every `pm` subcommand (dashboard, account, …), not introduced by this PR, but a read-only container path for read-only commands like `snapshot`/`dashboard` would restore the "planning does not update the DB" guarantee (source: codex) — `cmd/pm/main.go:54`, `internal/container/container.go`
- [ ] [debt] `pm snapshot` rounds with shopspring `.Round(0)`/`.Round(2)` (half-away-from-zero), whereas the removed `snapshot.py` used Python `round()` (banker's rounding). Real-DB output is currently byte-identical, but a value whose rounded digit sits exactly on `.5` (plausible for USD-converted amounts and 2-dp weights) would differ by 1 unit — a latent parity break. Half-away is arguably the better money rounding; decide whether exact snapshot.py parity matters and, if so, switch to `RoundBank` for value_krw/totals/weight_pct (source: review, confidence 48/low) — `cmd/pm/snapshot.go:205`
