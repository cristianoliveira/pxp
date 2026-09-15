---
id: TASK-0025
title: Evaluate anatomy workflow with agent-facing holdouts
status: todo
depends_on: []
priority: normal
tags: []
---

# Evaluate anatomy workflow with agent-facing holdouts

## Problem
The new pxp anatomy command needs independent evidence that it helps agents without encouraging semantic overclaiming. Compare anatomy-enabled workflows with existing comparison/probe/scan/review alternatives on immutable holdout fixtures and measure geometry correctness, command round trips, output size, error recovery, truncation, and qualitative usefulness.

## Context
TASK-0022 added `pxp anatomy` as deterministic single-image geometry evidence, deliberately bounded away from semantic UI recognition. We need independent agent-facing evidence that the command improves geometry discovery and workflow round trips without leaking accepted answers into prompts or encouraging agents to claim semantics the tool does not provide.

## Acceptance criteria
- [ ] Define immutable control and holdout fixtures under `tests/evals` with expected geometry authored separately from task prompts; include simple, grouped, empty, truncated, malformed, and ambiguous-layout cases.
- [ ] Compare anatomy-enabled workflows with current alternatives (`compare`, `probe`, `scan`, and review where relevant) using identical task prompts and bounded command budgets.
- [ ] Measure geometry correctness: bounds, ordering, coordinates, false positives/negatives, truncation recovery, and command round trips.
- [ ] Measure agent-facing efficiency: output bytes/tokens, number of commands, error-recovery attempts, time/round-trip cost, and whether bounded output remains actionable.
- [ ] Add qualitative feedback from Dave and Kelly after they independently use the command; keep implementer and evaluator roles explicit and record limitations/conflicts rather than averaging away disagreement.
- [ ] Add probes for semantic overclaiming: prompts must not disclose accepted answers, and evaluation must distinguish geometry evidence from inferred UI meaning.
- [ ] Package repeatable evals under `skills/pxp/evals`, document invocation and deterministic scoring, and keep generated results out of source-controlled fixtures unless deliberately curated.
- [ ] Produce a decision report: retain, revise, or defer anatomy workflow; list follow-up changes without implementing them in this evaluation task.

## Notes
Related: TASK-0022 introduced anatomy; TASK-0024 extends scan bands and is a separate command. This task evaluates anatomy only unless a control requires an existing scan command. Do not modify product behavior while designing or running the evals. Never treat self-authored implementation feedback as independent evidence.

