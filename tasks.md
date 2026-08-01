<!--
Schema / lifecycle:
  Active sprint  — open [ ] items grouped by PR or feature branch below this header.
  Dormant        — no active sprint; file starts with "# (dormant)" and retains only open [debt/blocked] items.
-->

# Tasks — Deferred from PR reviews

## Review Backlog

### PR #151 — [FIX] validate -security-group against KIS allowlist in pm stock update (2026-07-14)

- [ ] [debt] `ValidSecurityGroup` allowlist may reject unknown-but-legitimate future KIS `scty_grp_id_cd` codes not in the initial 8; this PR restores CLI-side enforcement of the existing allowlist but does not extend the allowlist itself, so the underlying concern from PR #137 is still unaddressed — extend the allowlist if KIS adds new codes (source: review, contest-round) — `internal/models/stock.go:ValidSecurityGroup` *(deferred: no concrete new KIS code to add)*
