---
id: TASK-0019
title: Replace square annotation markers with compact numbered badges
status: done
depends_on: []
priority: normal
tags: []
---

# Replace square annotation markers with compact numbered badges

## Problem
The review canvas currently renders a filled square marker for each annotation. Reviewers want a smaller visual treatment that identifies the note by number without adding a large shape over the evidence.

## Context
The review canvas currently draws a 22×22 filled square at every annotation anchor and places the note number inside it. The latest review-loop UX already exposes numbered note cards; the canvas marker can become a smaller, less obstructive visual anchor while retaining the mapping to the corresponding note.

## Acceptance criteria
- [x] Replace the filled square canvas marker with a compact numbered badge; use a small circular treatment unless visual review rejects it, with readable contrast for normal and selected states.
- [x] Keep the number mapped to the existing note ordering and make the anchor position unambiguous for points and rectangles.
- [x] Preserve rectangle outlines, selected-state distinction, source-view visibility, keyboard placement/editing, note-card navigation, and draft persistence.
- [x] Preserve a usable interaction target and accessible text mapping even if the visual badge is smaller; do not rely on color alone.
- [x] Make keyboard navigation first-class: every marker/card association is reachable in a predictable order, exposes an accessible name with its note number and source coordinates, supports Enter/Space activation, visibly restores focus after selection/edit/remove, and never traps focus.
- [x] Add or update browser coverage for marker shape/size, number-to-card mapping, selected state, keyboard-only navigation/activation/focus restoration, all source views, narrow viewport, and 200% zoom.
- [x] Capture before/after visual evidence and run focused plus full regression checks.

## Completion evidence
Implemented and merged in PR #38 at `8a8ef49`. The merged browser checks cover compact circular badges, selected state, number-to-card mapping, keyboard activation, source views, and responsive layouts. The persisted annotation contract and note ordering remain unchanged. CI passed before merge.

## Notes
Related: TASK-0017 and TASK-0018 established the current review-loop layout and numbered note-card mapping. Implementation was explicitly authorized. Avoid changing persisted annotation data or note numbering semantics.

