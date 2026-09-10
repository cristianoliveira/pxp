---
id: TASK-0003
title: Clarify command and report package names
status: done
depends_on: []
priority: low
tags: []
---

# Clarify command and report package names

## Problem
pixelperfectcmd and pixelperfectreport use inconsistent naming alongside pxp and capability packages; cli versus pixelperfectcmd obscures the runtime-helper boundary.

## Approach
Optional navigation cleanup; lower priority than TASK-0001 and TASK-0002.
- Rename `internal/pixelperfectcmd` to `internal/commands`.
- Rename `internal/pixelperfectreport` to `internal/report`.
- Keep `internal/cli` for process/runtime helpers and `cmd/pxp` as the executable boundary.
- Preview semantic references before applying renames. Update package declarations, imports, callers, documentation, module guides and test paths.

This task has no hard prerequisite. Prefer executing after workflow extraction to avoid overlapping moves, but do not model that preference as a dependency. Read the destination modules' existing guides before editing.

## Acceptance criteria
- [ ] New names consistently distinguish command orchestration, runtime helpers and HTML reporting.
- [ ] No stale runtime imports, package declarations or active documentation links reference old paths. Preserve historical plan references where appropriate.
- [ ] Check hardcoded paths, skill-drift tests and fixture-relative paths explicitly; do not rely only on compiler errors.
- [ ] `go build ./cmd/pxp` and affected command/report tests pass through the configured watcher; obtain fresh project verification.
- [ ] CLI executable name, command tree, public output contracts and report contents are unchanged.
- [ ] Diff is rename/import/documentation-only; no logic changes or new tests needed unless existing path coverage is missing.

## Non-goals
No top-level reorganization, `services/models/controllers` layers, merging of `cli` into commands, or renaming of `imagediff`/`imagecontext`.

## Completion
Commit with TASK-0003 in the message, then move this task to `plans/done` with status `done`.
