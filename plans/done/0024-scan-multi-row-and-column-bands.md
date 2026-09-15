---
id: TASK-0024
title: Scan multi-row and column bands
status: done
depends_on: []
priority: normal
tags: [cli, imagediff, scan]
---

# Scan multi-row and column bands

## Problem
A single-pixel scan is sensitive to antialiasing and exact coordinate choice. Agents need to inspect thicker borders, separators, text bands, and component interiors without issuing many unrelated scan commands or averaging away defects.

## Desired outcome
Extend `pxp scan` with an explicit band range. Keep every source row or column observable, then add a compact deterministic summary of patterns shared across the band. Single-row and single-column behavior remain the default.

Example direction:

```bash
pxp scan reference.png actual.png --rows 20:30
pxp scan reference.png actual.png --columns 40:48
```

## Contract decisions before implementation
- Define whether range end is inclusive or exclusive and expose normalized start/end in structured output. Prefer one convention shared with existing crop/bounds vocabulary.
- Do not average colors across the band by default. Averaging can hide local defects.
- Define “shared pattern” in pixel terms before adding a summary. If no useful deterministic summary survives differing run boundaries, return bounded per-line results first rather than inventing confidence.
- Define interaction with crops, `--limit`, `--full`, and existing `--row`/`--column` mutual exclusion. Band coordinates use the same comparison/cropped coordinate system as current scans.

## Acceptance criteria
- [ ] Scan a horizontal row range and vertical column range with symmetric syntax and output.
- [ ] Preserve the exact per-line color runs for both reference and actual images; no default averaging or semantic UI labels.
- [ ] Return query context, coordinate space, normalized range, `total`, `returned`, and `truncated`. Bounded output includes an exact `--full` recovery hint.
- [ ] Keep existing `--row`, `--column`, `--x`, and `--y` output and behavior backward-compatible.
- [ ] Reject empty, reversed, malformed, mixed-axis, and out-of-bounds ranges with exit code 2 and self-correcting examples. Validate before comparison work where possible.
- [ ] Produce byte-for-byte deterministic output for identical inputs and options.
- [ ] Document when a band scan helps: thick borders, separators, text/background bands, and verifying a region returned by `pxp anatomy`.
- [ ] Add focused domain tests for band extraction and command tests for parsing, bounds, crops, truncation, both axes, and all supported formats. Cover happy and failure paths before implementation.

## Non-goals and authorization
Do not infer UI elements, blur images, average colors by default, or change comparison metrics. This plan does not authorize implementation. The task complements `pxp anatomy`, but neither command depends on the other at runtime.
