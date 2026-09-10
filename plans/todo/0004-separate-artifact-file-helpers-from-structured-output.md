---
id: TASK-0004
title: Separate artifact file helpers from structured output
status: doing
depends_on: []
priority: low
tags: []
---

# Separate artifact file helpers from structured output

## Problem
Image analysis, annotations and reports import output for filesystem helpers, coupling file persistence to the package that owns JSON and TOON serialization.

## Approach
Boundary cleanup, independent of the command refactor. TASK-0005 builds on this artifact owner to separate image computation from I/O.
- Move `internal/output/file.go` and `file_test.go` into a narrow `internal/artifact` package.
- Preserve `CreateFile` and `WriteFile` signatures and behavior; update consumers in annotations, imagediff and report (currently pixelperfectreport).
- Retain printers and format selection in `internal/output`.
- Read existing module guides and document artifact ownership without creating a general-purpose filesystem utility package.

## Acceptance criteria
- [ ] Inspect references before moving helpers and update all callers; artifact-only consumers no longer import `output`.
- [ ] Preserve parent-directory creation, directory/file permissions, overwrite behavior and returned errors.
- [ ] Move existing helper tests unchanged. Add missing failure-path or permissions characterization tests before changing implementation, if needed.
- [ ] Annotation, image artifact and HTML report tests pass; command smoke tests retain output paths and artifact contents.
- [ ] `artifact` depends only on necessary standard-library filesystem facilities; it introduces no serialization dependency or new abstraction.
- [ ] Obtain fresh watcher verification and update module guides to match the dependency direction.

## Non-goals
No atomic-write feature, permission-policy change, I/O interface framework, or broad removal of image decoding/writing APIs from imagediff.

## Completion
Commit with TASK-0004 in the message, then move this task to `plans/done` with status `done`.
