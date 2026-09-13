---
id: TASK-0016
title: Refactor first cyclomatic complexity hotspot
status: done
depends_on: []
priority: normal
tags: []
---

# Refactor first cyclomatic complexity hotspot

## Problem
The official roadmap needs a bounded first implementation slice so complexity can decrease through tested, cohesive refactors instead of remaining a policy document. The first hotspot must be reduced without changing observable CLI, image-evidence, or review-loop behavior.

## Context
This task operationalizes `docs/complexity-refactoring-roadmap.md` and follows TASK-0015. It is a planning task only until implementation is explicitly authorized.

The initial candidate should be selected from the current hotspot inventory after measuring the clean target branch. Prefer a function with a clear cohesive responsibility, strong existing behavior coverage, and a bounded extraction boundary. Do not assume the previous validation slice remains the next candidate without re-measuring.

## Proposed delivery plan
1. Capture baseline complexity, function length, package ownership, and focused/full test coverage for the selected function.
2. Add characterization tests for success, boundary, and failure behavior before changing structure.
3. Extract cohesive decisions or adapters; avoid speculative abstractions and metric gaming through shallow wrappers.
4. Verify focused tests, full tests, lint, canonical quality checks, and before/after metrics.
5. Record residual complexity, changed contracts, risk, rollback boundary, and whether the next ratchet stage is justified.

## Acceptance criteria for implementation
- [x] Selected hotspot and selection rationale are recorded before coding.
- [x] Characterization tests cover existing observable behavior and pass before and after the refactor.
- [x] Complexity and function length improve measurably without changing public CLI/output/image/review-loop behavior.
- [x] No new or modified function exceeds the roadmap's changed-code target of complexity 10 without an explicit documented exception.
- [x] Focused and full verification pass; quality evidence includes coverage and remaining hotspot/exemption trend.
- [x] Refactor is one cohesive slice with a clear rollback boundary and no unrelated cleanup.
- [x] Roadmap/inventory records the result and names the next candidate or explains why another ratchet is not yet safe.

## Implementation slice

Selected `internal/commands/compare.go:runComparisonCommand` after measuring the
clean `origin/main` (`5a70828`). It was the highest hotspot: complexity 40 and
209 lines. The command is high value because it sequences comparison, output,
reports, optional context, region evidence, and validation; existing command and
smoke tests cover its observable CLI and artifact contract. The next candidates
were `review.NewSessionWithContext` (33 / 166 lines),
`imagediff.DecodedImages.Compare` (28 / 152), and
`imagecontext.LoadProviderConfig` (24 / 80).

Characterization coverage was added before the runtime refactor in
`TestDiffImageCommandPreservesAnalysisArtifacts`, covering mask, overlay, offset,
movement, region grouping/filtering, and structured regions. The focused test
passed before source changes.

The refactor keeps `runComparisonCommand` as orchestration (complexity 7 / 68
lines) and extracts cohesive option parsing, annotation loading, artifact
materialization, region processing/metrics, and final output phases. All new or
modified functions are at complexity <=10; the highest is
`writeComparisonArtifacts` at 9. Existing validation hotspots remain tracked,
and no threshold changed. Rollback is the single implementation commit; the
characterization test remains valid on either side.

Post-change trend: `runComparisonCommand` 40 -> 7 and 209 -> 68 lines; package
quality remains under the existing caps. The next candidate is
`review.NewSessionWithContext`; do not begin it as part of this slice.

## Authorization boundary
Implementation was explicitly authorized for TASK-0016. TASK-0009 guardrails
and TASK-0015 roadmap remain related quality work; this task does not alter
thresholds.

