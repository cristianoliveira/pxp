---
id: TASK-0022
title: Add pxp anatomy command for single-image element bounds
status: doing
depends_on: []
priority: normal
tags: [cli, imagediff]
---

# Add pxp anatomy command for single-image element bounds

## Problem
Agents need absolute element geometry from one screenshot (text-line bands, icon/button boxes) but every pxp answer is comparative; the blank-canvas workaround collapses into one merged region, which forced a throwaway Go helper during the card-alert work.

## Context
Design agreed with the user on 2026-09-14; full direction in `docs/anatomy-vision.md`.

Approach:
- Evidence in `internal/imagediff`: background detection (most frequent image color, deterministic tie-break, `--background` override) + deterministic connected components; no gap grouping by default (`--group N` opt-in).
- Command in `internal/commands/anatomy.go`, wired in `command.go`; output via `internal/output.Printer` (TOON default, `--json` compat).
- Default fields: bounds x,y,w,h; pixels; density; dominant color. Reading-order sort. `total`/`returned`/`truncated` + `--full` escape, matching scan's bounded-output contract.
- Flags: `--threshold 8`, `--background`, `--group 0`, `--min-pixels 4`, `--limit`, `--full`, `--format`.

## Acceptance criteria
- [ ] `pxp anatomy skills/pxp/fixtures/card-alert.png` yields the fixture's known elements (icon, title band, metadata band, 3 description lines, 2 buttons, ellipsis) as separate regions
- [ ] Solid-color image returns an explicit empty result naming the query context (not silent success)
- [ ] `--group` merges only within the given gap; same invocation is byte-for-byte deterministic
- [ ] Truncation path sets `truncated` and the `--full` hint; usage errors exit 2 with named input
- [ ] Contract documented in `docs/command-contracts.md`; no presentation logic in imagediff

## Notes

