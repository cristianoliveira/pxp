---
id: TASK-0020
title: Remove static HTML report flag in favor of review
status: todo
depends_on: []
priority: normal
tags: []
---

# Remove static HTML report flag in favor of review

## Problem
The primary interactive workflow is split between a static `--report` HTML artifact and the blocking `pxp review` loop. Maintaining both as review entry points creates ambiguity for agents and humans; interactive review should have one obvious command.

## Desired outcome
`pxp review` is the single supported human-review workflow. Comparison remains non-interactive and machine-readable. Agents do not generate a static HTML report when they need human feedback.

## Current evidence
- Root comparison declares `--report` in `internal/commands/command.go` and writes through `internal/report` from `internal/commands/compare.go`.
- Report behavior and path collisions have command tests in `internal/commands/compare_test.go`.
- README and `skills/pxp/SKILL.md` still recommend `--report` for interactive inspection.
- `pxp review` already persists immutable screenshots, overlays, context, and feedback while owning the blocking human decision lifecycle.

## Migration plan
1. Inventory documented and code-level consumers of `--report`; determine whether any release or external automation compatibility window is required.
2. Remove `--report` from comparison help and execution. If compatibility policy requires staged removal, first return a clear deprecation diagnostic that points interactive users to `pxp review`; define the removal release rather than leaving indefinite dual behavior.
3. Update README, command contracts, skills, examples, and evaluation expectations. Lead with `pxp review` for human review and plain structured comparison for automation.
4. Remove report-specific command wiring, validation, and tests. Remove `internal/report` only after usage evidence proves it has no remaining owner or consumer; do not conflate flag removal with an unverified package deletion.
5. Verify help, invalid legacy invocation, review loop, machine-readable comparison, masks/overlays, and skill behavior.

## Acceptance criteria
- [ ] Interactive documentation and agent skills use `pxp review`, wait for the explicit human decision, and do not recommend `--report` as a review fallback.
- [ ] Root comparison help no longer advertises `--report`; the chosen compatibility behavior for legacy use is explicit, tested, and provides a self-correcting migration to `pxp review` rather than silently ignoring the flag.
- [ ] Removing report output does not change comparison metrics, structured stdout, exit codes, masks, overlays, crops, regions, thresholds, or validation gates.
- [ ] `pxp review` still provides Reference / Current / Overlay, immutable round evidence, context, annotations, explicit Submit/Approve, foreground wait, and server cleanup.
- [ ] Tests cover command help, legacy `--report` behavior, non-interactive comparison, and a blocking review completion. Remove obsolete report-render assertions rather than replacing them with markup-string tests.
- [ ] Search and public docs contain no stale command examples or claims that comparison generates HTML reports. Uses of the word “report” for diagnostics or human summaries remain valid and are not blindly removed.
- [ ] Release notes call out the breaking CLI change and migration command. If semantic-version policy requires deprecation first, record owner and removal milestone.
- [ ] `internal/report` is deleted only if a fresh usage/ownership check shows it is unreachable after flag removal; otherwise document its remaining responsibility.

## Non-goals and authorization
Do not remove `pxp` comparison, overlays, masks, structured metrics, or saved review-round artifacts. Do not add a replacement export format in this task. This plan does not authorize implementation; removing a public flag is a breaking change and requires explicit implementation approval after compatibility review.

