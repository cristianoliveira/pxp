---
id: TASK-0001
title: Split command implementation by workflow
status: done
depends_on: []
priority: high
tags: []
---

# Split command implementation by workflow

## Problem
command.go mixes command wiring, comparison, inspection, input preparation and CSV rendering in 1,390 lines, making changes difficult to navigate and review.

## Approach
Keep the current package and signatures. Move existing declarations into focused files:
- `command.go`: root command construction and shared command wiring.
- `compare.go`: comparison workflow, validation gates and artifact orchestration.
- `probe.go` and `scan.go`: respective commands and existing inspection helpers.
- `inputs.go`: crop/metadata preparation and coordinate mapping.
- `inspection_output.go`: inspection result presentation and CSV writers.

Keep small shared helpers beside their owner; do not create a generic helpers package. Split `command_test.go` along the same workflow boundaries where practical, retaining test names and assertions. Read the module AGENTS.md before implementation. Paths here refer to the current names; TASK-0003 may rename the package independently.

## Acceptance criteria
- [ ] Each file has one clear workflow or shared responsibility; root wiring no longer contains image algorithms or CSV writers.
- [ ] Function signatures, visibility, execution order and behavior remain unchanged. No new exported API or packages.
- [ ] Existing tests move without weakening assertions. This is a mechanical refactor, not a behavior change.
- [ ] Command tests and smoke tests pass; help, flags, JSON/TOON/CSV output, errors, exit codes and artifact paths remain unchanged.
- [ ] Capture baseline verification before moving code and obtain fresh watcher verification afterward. Use watcher target discovery rather than guessing targets.
- [ ] Review the diff for accidental deletions and update module landmarks if function locations change.

## Non-goals
No package renames, algorithm optimizations, new interfaces, or movement of algorithms across packages. TASK-0002 owns the latter.

## Completion
Commit with TASK-0001 in the message, then move this task to `plans/done` with status `done`.
