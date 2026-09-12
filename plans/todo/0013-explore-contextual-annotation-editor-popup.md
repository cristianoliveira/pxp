---
id: TASK-0013
title: Explore contextual annotation editor popup
status: todo
depends_on: []
priority: normal
tags: []
---

# Explore contextual annotation editor popup

## Problem
The permanent Add annotation form occupies feedback-panel space and separates note entry from the selected evidence. The human asked whether that area could become a popup; the interaction and accessibility tradeoffs need a product decision before implementation.

## Evidence and provenance
- Source: `.tmp/task0011-proof/round-2/round-1218538722/feedback.json` (local artifact; exact relevant feedback preserved here).
- Session: `review-3aa7300cd4a277bb-17c907d5bd3ee437`; round: 2; decision: `submitted`.
- Annotation `note-0002`: image `actual`, rectangle x=958, y=174, width=273, height=191, original pixels.
- Exact note: “Can we change this and present as an popup instead? ”
- Actual screenshot: 1280×1140; SHA-256 `f2ba9203b5ffa3a0d5819a7ddffe1a9d037a328a3e08719494726b4354617271`.
- Exact general note: “Overall, looks good, but I added this one. You can add it as plans to improve for later developments. ”

Inspected screenshot and `internal/review/frontend/index.html`: the marked area covers the feedback heading and Add annotation form (Type and Note), not the general-note or decision buttons. Working interpretation is a contextual annotation editor; confirm exact scope before removing any persistent feedback UI.

## Discovery alternatives and decision criteria
Compare a contextual nonmodal popover, an accessible modal editor, and the current persistent form. Sketch how each opens after pointer and keyboard placement and how existing-note editing works. Decide with the human whether reduced panel clutter and proximity to evidence outweigh obscured pixels, focus transitions, and discoverability costs. A popup is a proposal, not an approved solution.

## Acceptance for design selection and later implementation
- [ ] Present a small interaction mock and identify exactly which controls move. Keep general feedback and Submit/Approve persistently reachable unless explicitly agreed otherwise.
- [ ] Both keyboard and pointer can open the editor, enter/edit a note, save, and cancel. No hover-only trigger; moving the pointer away does not lose text.
- [ ] Editor exposes a meaningful accessible name and source-view/geometry context. Focus entry, Tab order, Escape cancellation, and return to the invoking canvas/list control are deterministic. Modal focus is contained only while genuinely modal; nonmodal content must not claim modal semantics.
- [ ] Saving preserves original-image geometry, source view, and stable note identity. Canceling a new annotation creates no committed note; canceling an edit preserves the previous note and unrelated draft feedback.
- [ ] Define reopen/recovery for unsaved editor input; browser closure and outside clicks must not silently commit or discard it. Preserve pending-review and confirmation boundaries.
- [ ] Popup placement remains usable at narrow widths and 200% zoom; avoid obscuring the annotated target or make the evidence readily inspectable. No reliance on color alone or inaccessible small targets.
- [ ] Keyboard/screen-reader smoke and browser interaction checks cover creation/edit/save/cancel, view switching, focus restoration, geometry persistence, and reopen recovery. Document evidence and remaining limitations.

## Scope and status
Planning/discovery only; no runtime authorization. Independent of TASK-0012 view navigation and distinct from TASK-0011 automatic handoff proof. Follow `docs/review-loop-vision.md`: preserve keyboard completeness, accessibility, visual comfort, and feedback context. A popup must earn its complexity by improving those outcomes.

