---
id: TASK-0018
title: Add compact pin note cards to review loop
status: done
depends_on: []
priority: normal
tags: [review, ux]
---

# Add compact pin note cards to review loop

## Problem
Pinned feedback is difficult to scan because notes are not presented in a compact, location-preserving card treatment.

## Context
Feedback from review session `review-53ae0a1ed5a45014-d39e11712cc3c70c`.

## Acceptance criteria
- [x] Define the 22px target precisely: the on-canvas marker remains a 22px visual anchor while the associated note card uses a readable minimum size; do not force text into a 22px box.
- [x] Render each pin with a stable marker-to-card relationship; the card shows source view, original coordinates, and note text without obscuring the anchor or evidence.
- [x] Define wrapping/truncation for long and multiline notes, with an accessible way to read the full text; cards remain usable at narrow widths and 200% zoom.
- [x] Preserve keyboard access, annotation creation/edit/removal, focus restoration, source navigation, and draft persistence.
- [x] Expose marker/card association and expanded state programmatically; avoid hover-only access and ensure contrast/focus visibility.
- [x] Add browser coverage for compact pin-note presentation, long text, keyboard interaction, responsive layout, and source-location navigation.

## Implementation contract

- Point and rectangle anchors retain a numbered 22×22 source-pixel marker on the
  evidence canvas; note text is never constrained to that marker size.
- The feedback list renders newest-first cards with the matching marker number,
  source view, original coordinates, optional rectangle dimensions, and note
  text. Long or multiline notes remain readable through an explicit
  `Read full note` control with programmatic expanded state.
- Selecting a card navigates to its source view and anchor; keyboard annotation
  editing/removal, source shortcuts, LIFO ordering, and local drafts remain
  unchanged.
- Browser coverage exercises long/multiline notes, marker/card semantics,
  keyboard expansion, source navigation, and narrow/large-text layouts.

## Verification

Independent QA PASSed exact commit `8ad27a5`. The browser suite, Go tests,
race, vet, quality gate, and diff check passed. Manual AT inspection confirmed
zero nested interactive buttons; card selection and full-note expansion are
sibling controls with correct `aria-controls`/`aria-expanded` state. Desktop and
320px/200% screenshots show readable cards with no horizontal overflow.

A combined foreground `pxp review --open --json` loop was completed against the
combined TASK-0017/TASK-0018 runtime. The reviewer explicitly submitted a pin,
note, and general note; persisted feedback has `decision: submitted`. No
approval was fabricated. Evidence: `/tmp/task0018-final-human-loop/` and
`/tmp/task0018-browser-proof-r2/`.

## Notes
General feedback: “Can we have the pins in a little 22 box with the notes of the pins, if it's possible?”

