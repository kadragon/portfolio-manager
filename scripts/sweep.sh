#!/bin/bash
# sweep.sh — Automated garbage collection for portfolio-manager harness
# Usage:
#   bash scripts/sweep.sh              # full sweep
#   bash scripts/sweep.sh --quick      # lint only
#
# Trigger policy: manual. Run between features or before PRs.

set -euo pipefail

SCRIPTS_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

RED='\033[0;31m'
YELLOW='\033[0;33m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

FINDINGS=()
QUICK_MODE=false
[[ "${1:-}" == "--quick" ]] && QUICK_MODE=true

cd "$PROJ_DIR"

echo -e "${CYAN}=== Sweep ===${NC}"
echo -e "  Date: $(date '+%Y-%m-%d %H:%M')"

# ── 1. Lint scan ─────────────────────────────────────────────
echo -e "${CYAN}[1/5] Lint scan (ruff)...${NC}"
if ruff_output=$(uv run ruff check src tests 2>&1); then
    echo -e "  ${GREEN}ruff clean${NC}"
else
    echo "$ruff_output" | tail -20
    FINDINGS+=("[lint] ruff reported violations — run 'uv run ruff check --fix src tests'")
fi

$QUICK_MODE && { echo "Quick mode — done."; exit ${#FINDINGS[@]}; }

# ── 2. Type check ───────────────────────────────────────────
echo -e "${CYAN}[2/5] Type check (pyright)...${NC}"
if pyright_output=$(uv run pyright 2>&1); then
    echo -e "  ${GREEN}pyright clean${NC}"
else
    echo "$pyright_output" | tail -10
    FINDINGS+=("[types] pyright reported errors")
fi

# ── 3. Golden principle spot-check ───────────────────────────
echo -e "${CYAN}[3/5] Golden principles...${NC}"
principle_issues=0

# Principle 1/3: Layer boundaries — run the dedicated arch test
if ! uv run pytest tests/arch/ -q --no-cov >/dev/null 2>&1; then
    FINDINGS+=("[arch] tests/arch/ failed — layer boundary or dependency direction violation")
    principle_issues=$((principle_issues + 1))
fi

# Principle 4: Secrets — bandit
if ! uv run bandit -q -r src >/dev/null 2>&1; then
    FINDINGS+=("[security] bandit flagged findings — run 'uv run bandit -r src'")
    principle_issues=$((principle_issues + 1))
fi

# Principle 2 (KIS live-test marking) is a convention — static detection
# produces too many false positives because unit tests mock httpx. Rely on
# author discipline + code review; reconsider if we add a dedicated
# tests/live/ directory.

[[ $principle_issues -eq 0 ]] && echo -e "  ${GREEN}All principles OK${NC}"

# ── 4. Harness freshness ────────────────────────────────────
echo -e "${CYAN}[4/5] Harness freshness...${NC}"
harness_issues=0

# Check that files referenced in AGENTS.md exist
if [[ -f "AGENTS.md" ]]; then
    while IFS= read -r doc; do
        [[ -z "$doc" ]] && continue
        if [[ ! -f "$doc" ]]; then
            FINDINGS+=("[harness] AGENTS.md references missing file: $doc")
            harness_issues=$((harness_issues + 1))
        fi
    done < <(grep -oE 'docs/[a-zA-Z0-9_./-]+\.(md|txt)' AGENTS.md | sort -u)
fi

# CLAUDE.md must be the @AGENTS.md pointer
if [[ -f "CLAUDE.md" ]] && [[ "$(tr -d '[:space:]' < CLAUDE.md)" != "@AGENTS.md" ]]; then
    FINDINGS+=("[harness] CLAUDE.md should contain only '@AGENTS.md'")
    harness_issues=$((harness_issues + 1))
fi

# AGENTS.md size band (target <=100, warn >200)
agents_lines=$(wc -l < AGENTS.md 2>/dev/null || echo 0)
if (( agents_lines > 200 )); then
    FINDINGS+=("[harness] AGENTS.md is $agents_lines lines — exceeds 200-line warning threshold")
    harness_issues=$((harness_issues + 1))
fi

[[ $harness_issues -eq 0 ]] && echo -e "  ${GREEN}Harness references valid${NC}"

# ── 5. Summary ──────────────────────────────────────────────
echo ""
if [[ ${#FINDINGS[@]} -eq 0 ]]; then
    echo -e "${GREEN}=== Sweep clean ===${NC}"
    exit 0
fi

echo -e "${YELLOW}=== ${#FINDINGS[@]} finding(s) ===${NC}"
for f in "${FINDINGS[@]}"; do echo "  $f"; done

if [[ -f "backlog.md" ]]; then
    {
        echo ""
        echo "## Sweep $(date '+%Y-%m-%d %H:%M')"
        for f in "${FINDINGS[@]}"; do
            echo "- [ ] $f"
        done
    } >> backlog.md
    echo -e "${GREEN}Appended ${#FINDINGS[@]} item(s) to backlog.md${NC}"
fi

exit 1
