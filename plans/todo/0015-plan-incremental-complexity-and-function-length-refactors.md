---
id: TASK-0015
title: Plan incremental complexity and function-length refactors
status: todo
depends_on: []
priority: normal
tags: []
---

# Plan incremental complexity and function-length refactors

## Problem
The repository guardrails currently allow legacy functions up to cyclomatic complexity 40 and 203 lines, which prevents regressions but does not create a path toward simpler code. We need an evidence-based ratchet plan that lowers limits safely through focused refactors.

## Context
TASK-0009 established compatibility-oriented whole-repository caps: cyclomatic complexity 40 and function length 203 lines / 127 statements. These are ceilings, not quality targets. The next phase should lower them in measured increments while preserving behavior and avoiding broad speculative rewrites.

## Acceptance criteria
- [ ] Inventory current cyclomatic and function-length hotspots with package, symbol, metric, ownership, and test evidence.
- [ ] Group hotspots by cohesive responsibility and rank refactors by risk and user value; do not optimize metrics by scattering logic or adding shallow wrappers.
- [ ] Define a ratchet sequence (for example, complexity 40 -> 30 -> 20 -> 15 -> 10) with explicit entry criteria, rollback conditions, and baseline/exemption handling.
- [ ] Establish a changed-code policy requiring new or modified functions to target complexity <=10 and reasonable function length, while legacy exceptions are tracked and trend downward.
- [ ] For each increment, add characterization/property tests before refactoring and preserve observable CLI, output, image, and review-loop behavior.
- [ ] Keep quality checks deterministic and runnable locally/CI; report trend metrics and remaining exemptions after each increment.
- [ ] Identify the first small refactor slice suitable for separate implementation authorization; this planning task must not change runtime code or thresholds.

## Notes
Related: TASK-0009 established the current guardrails. This task plans later refactors only; no threshold change is authorized by creating it. Prefer one cohesive hotspot per implementation task and require focused plus full regression evidence.

