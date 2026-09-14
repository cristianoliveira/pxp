---
id: TASK-0020
title: Remove static HTML report flag in favor of review
status: done
depends_on: []
priority: normal
tags: []
---

# Remove static HTML report flag in favor of review

## Problem
The primary interactive workflow is split between a static `--report` HTML artifact and the blocking `pxp review` loop. Maintaining both as review entry points creates ambiguity for agents and humans; interactive review should have one obvious command.

## Desired outcome
`pxp review` is the single supported human-review workflow. Comparison remains non-interactive and machine-readable. Agents do not generate a static HTML report when they need human feedback.

## Initial evidence
- Root comparison declared `--report` in `internal/commands/command.go` and wrote through `internal/report` from `internal/commands/compare.go`.
- Report behavior and path collisions had command tests in `internal/commands/compare_test.go`.
- README and `skills/pxp/SKILL.md` recommended `--report` for interactive inspection.
- `pxp review` already persisted immutable screenshots, overlays, context, and feedback while owning the blocking human decision lifecycle.

## Compatibility decision

Inventory on 2026-09-14 found `--report` consumers in the comparison command
flag/options/writer, command and smoke tests, the root and command README files,
`docs/command-contracts.md` references, `skills/pxp/SKILL.md`, and the upload
panel evaluation verifier. The `internal/report` package is only reachable
through the comparison writer; no release workflow or external automation in
this repository consumes its HTML output.

This is a deliberate breaking removal. Legacy `--report` invocation will fail
with a usage diagnostic that points to `pxp review <reference.png> <actual.png>`
for interactive review and plain comparison for structured machine output. The
flag will not be silently ignored or retained as an indefinite deprecation.

## Migration plan
1. Inventory documented and code-level consumers of `--report`; determine whether any release or external automation compatibility window is required.
2. Remove `--report` from comparison help and execution. If compatibility policy requires staged removal, first return a clear deprecation diagnostic that points interactive users to `pxp review`; define the removal release rather than leaving indefinite dual behavior.
3. Update README, command contracts, skills, examples, and evaluation expectations. Lead with `pxp review` for human review and plain structured comparison for automation.
4. Remove report-specific command wiring, validation, and tests. Remove `internal/report` only after usage evidence proves it has no remaining owner or consumer; do not conflate flag removal with an unverified package deletion.
5. Verify help, invalid legacy invocation, review loop, machine-readable comparison, masks/overlays, and skill behavior.

## Acceptance criteria
- [x] Interactive documentation and agent skills use `pxp review`, wait for the explicit human decision, and do not recommend `--report` as a review fallback.
- [x] Root comparison help no longer advertises `--report`; legacy use is explicitly tested and provides a self-correcting migration to `pxp review` rather than silently ignoring the flag.
- [x] Removing report output does not change comparison metrics, structured stdout, exit codes, masks, overlays, crops, regions, thresholds, or validation gates.
- [x] `pxp review` still provides Reference / Current / Overlay, immutable round evidence, context, annotations, explicit Submit/Approve, foreground wait, and server cleanup.
- [x] Tests cover command help, legacy `--report` behavior, non-interactive comparison, and a blocking review completion; obsolete report-render assertions were removed.
- [x] Search and public docs contain no stale command examples or claims that comparison generates HTML reports. Uses of the word “report” for diagnostics or human summaries remain valid.
- [x] Release notes call out the breaking CLI change and migration command.
- [x] `internal/report` was deleted only after a fresh usage/ownership check showed it was unreachable after flag removal.

## Completion evidence

- Implementation merged to `origin/main` at `4399bd1` via PR #40; cleanup follow-up was `905cc69`.
- Kelly QA PASS on exact `905cc69`; report: `.tmp/reports/13-09-26/task-0020-qa.md`.
- Watcher generation 36 passed format, vet, lint, full tests, review browser e2e, and install; quality gate passed at 79.6% coverage.
- GitHub CI passed lint and macOS/Ubuntu tests.
- Compatibility policy: deliberate breaking removal. `--report` is absent from help; `--report path` and `--report=path` return usage exit code 2 with a migration diagnostic to `pxp review`, never silently ignored.
- Exact parity checks confirmed structured comparison metrics and mask/overlay bytes remain unchanged after path normalization.
- Board closed from clean latest `origin/main` (`4399bd1`) with this transition commit.

## Non-goals and authorization
Do not remove `pxp` comparison, overlays, masks, structured metrics, or saved review-round artifacts. Do not add a replacement export format in this task. This plan does not authorize implementation; removing a public flag is a breaking change and requires explicit implementation approval after compatibility review.
