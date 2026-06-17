# Evaluation Criteria

## Sprint Contract

Before implementing, agree on "done" in one sentence per criterion:

```
Sprint Contract — [feature name]

Correctness:   [what passing looks like — specific test or command]
Coverage:      [minimum test coverage or scenario list]
Layer safety:  [arch tests pass — go test ./internal/arch/... exits 0]
No regressions:[full test suite green — go test ./... exits 0]
```

Both generator and evaluator must agree before implementation starts. If no evaluator, write the contract solo and treat it as a hard gate.

## Evaluation Rubric

For each criterion, score 1, 3, or 5 (odd only — even values are undefined):

| Score | Meaning |
|-------|---------|
| 5 | Excellent — exceeds expectations |
| 3 | Acceptable — meets the bar |
| 1 | Broken — criterion fails |

**Pass threshold:** All criteria ≥ 3.

## Standard Criteria (portfolio-manager)

### 1. Correctness (weight: 3)
- Score 5: All specified behaviors work; no edge cases missed in the implementation scope
- Score 3: Core behaviors work; minor edge cases may be missing but documented
- Score 1: Core behavior broken or missing

How to test: `go test ./...` green; manually verify the new behavior in the running app.

### 2. Layer safety (weight: 3)
- Score 5: `go test ./internal/arch/` passes; no direct DB imports in `web/` or `services/`
- Score 3: Same as 5 (this is binary — arch tests either pass or fail)
- Score 1: Arch tests fail

How to test: `go test ./internal/arch/ -v`

### 3. Test coverage (weight: 2)
- Score 5: New behavior has unit tests; edge cases covered; integration test uses `//go:build integration` tag if KIS API involved
- Score 3: Core behavior tested; edge cases may be missing
- Score 1: No tests for new behavior

How to test: `make go-cover`

### 4. No regressions (weight: 2)
- Score 5: Full suite green
- Score 3: Same as 5 (regressions are binary)
- Score 1: Existing tests broken by change

How to test: `go test -failfast ./...`

## Evaluator Protocol

1. Read Sprint Contract before looking at code
2. Grade each criterion independently — write score + evidence before moving to next
3. Do not rationalize — a bug is a bug even if it's "minor"
4. If score < 3 on any criterion: fail the sprint, document what's missing
5. Re-evaluate only after generator fixes the gap — not after explanation

## Common Failure: Self-Deception

Agent finds a gap → rationalizes ("it's an edge case", "low priority") → grades 3 anyway.

Countermeasure: Write the evidence sentence first (`"Test X shows..."`) then assign the score. If you cannot write evidence, the score is 1.
