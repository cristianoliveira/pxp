---
id: TASK-0016
title: Refactor first cyclomatic complexity hotspot
status: doing
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
- [ ] Selected hotspot and selection rationale are recorded before coding.
- [ ] Characterization tests cover existing observable behavior and pass before and after the refactor.
- [ ] Complexity and function length improve measurably without changing public CLI/output/image/review-loop behavior.
- [ ] No new or modified function exceeds the roadmap's changed-code target of complexity 10 without an explicit documented exception.
- [ ] Focused and full verification pass; quality evidence includes coverage and remaining hotspot/exemption trend.
- [ ] Refactor is one cohesive slice with a clear rollback boundary and no unrelated cleanup.
- [ ] Roadmap/inventory records the result and names the next candidate or explains why another ratchet is not yet safe.

## Authorization boundary
No implementation is authorized by creating this task. Ask the user before moving it to doing or assigning implementation. TASK-0009 guardrails and TASK-0015 roadmap remain related quality work; this task must not silently alter their thresholds.

