---
id: TASK-0002
title: Move probe and scan measurements into imagediff
status: done
depends_on: [TASK-0001]
priority: normal
tags: []
---

# Move probe and scan measurements into imagediff

## Problem
Deterministic probe and scan algorithms live in the command package, coupling reusable image evidence to CLI policy and output shapes.

## Evidence and scope
Originally in `internal/pixelperfectcmd/command.go`: `scanPNGRuns`, `probeImages`, `probePNGColor`, and `colorFromImage`. TASK-0001 first isolates these workflows so extraction can be reviewed separately from file reorganization.

## Approach
1. Characterize current CLI behavior before extraction, including colors, deltas, run boundaries and coordinate mapping.
2. Write direct failing tests for a small image-based measurement API in `internal/imagediff` (`probe.go`, `scan.go` and colocated tests).
3. Move deterministic measurements there. Prefer decoded image inputs; keep file lifecycle and usage-error translation explicit at the command boundary.
4. Map evidence into existing command result envelopes. Keep Cobra flags, inspection limits, input/crop coordinates, hex/CSV presentation and usage wording in commands.

Read module guides before implementation. Do not export CLI DTOs merely to make the move compile. Keep any decode-caching or color-semantics change out of this refactor.

## Acceptance criteria
- [ ] Direct deterministic tests cover known colors and deltas, identical pixels, alpha behavior, horizontal/vertical runs, first/last positions and invalid measurement bounds.
- [ ] Command tests cover malformed input, mismatched dimensions, out-of-bounds requests and crop-origin mapping. Existing decoding failures remain covered.
- [ ] Preserve current color conversion semantics, run endpoints/order, serialized field names, truncation metadata, error classifications and exit codes.
- [ ] `imagediff` imports neither Cobra nor `cli`; measurement APIs do not expose command envelopes.
- [ ] Record red/green evidence and focused coverage for the new algorithms. Run command/smoke regression checks and obtain fresh watcher verification.
- [ ] Update module guidance to identify the new measurement owner and entry points.

## Non-goals
No new image-analysis package, CLI contract changes, performance project, or changes to comparison thresholds.

## Completion
Commit with TASK-0002 in the message, then move this task to `plans/done` with status `done`.
