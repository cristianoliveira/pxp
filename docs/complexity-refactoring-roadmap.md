# Cyclomatic complexity reduction roadmap

## Purpose

The current complexity limits protect the repository from regressions, but they are compatibility ceilings, not a design target. This roadmap makes the reduction effort explicit and measurable without trading readability or behavior for a metric.

## Baseline

- Current repository ceiling: cyclomatic complexity **40** per function.
- Current function-length ceiling: **203 lines / 127 statements**.
- Existing hotspots remain tracked as legacy exceptions until they are refactored.
- The first refactor slice should be one cohesive responsibility with characterization tests before extraction.

## Ratchet

| Stage | Maximum for legacy code | New or changed code | Exit evidence |
|---|---:|---:|---|
| Current | 40 | ≤10 target | Baseline inventory and tests |
| 1 | 30 | ≤10 target | All >30 hotspots refactored or explicitly exempted |
| 2 | 20 | ≤10 target | Focused tests and full regression pass |
| 3 | 15 | ≤10 target | Exemptions shrink and trend report is clean |
| Target | 10 | ≤10 | No unreviewed exceptions |

A stage changes only after its exit evidence is reviewed. Do not lower a limit merely to make a dashboard green; do not scatter logic into shallow wrappers to reduce a score.

## Work sequence

1. Inventory hotspots with package, symbol, complexity, length, ownership, user impact, and test evidence.
2. Rank by cohesive responsibility and regression risk. Prefer one hotspot per implementation task.
3. Add characterization or property tests for observable behavior before refactoring.
4. Extract cohesive decisions, validation, and adapters while preserving CLI output, image evidence, review-loop decisions, and error contracts.
5. Run focused tests, full tests, lint, and the canonical quality check.
6. Record before/after metrics, changed behavior evidence, remaining exceptions, and rollback notes.
7. Propose the next ratchet only after the current stage is stable.

## Governance

This is an engineering quality roadmap, not an automatic mandate to refactor every hotspot. Each implementation task must state its selected slice, tests, risk, and rollback boundary. Threshold changes require an explicit review of baseline and exemptions. Existing behavior remains the contract unless a separate product decision changes it.

The roadmap complements `docs/quality-guardrails.md` and TASK-0015. The guardrails define what currently fails; this document defines how the codebase can safely improve beyond those compatibility limits.
