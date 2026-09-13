---
id: TASK-0018
title: Add compact pin note cards to review loop
status: doing
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
- [ ] Define the 22px target precisely: the on-canvas marker remains a 22px visual anchor while the associated note card uses a readable minimum size; do not force text into a 22px box.
- [ ] Render each pin with a stable marker-to-card relationship; the card shows source view, original coordinates, and note text without obscuring the anchor or evidence.
- [ ] Define wrapping/truncation for long and multiline notes, with an accessible way to read the full text; cards remain usable at narrow widths and 200% zoom.
- [ ] Preserve keyboard access, annotation creation/edit/removal, focus restoration, source navigation, and draft persistence.
- [ ] Expose marker/card association and expanded state programmatically; avoid hover-only access and ensure contrast/focus visibility.
- [ ] Add browser coverage for compact pin-note presentation, long text, keyboard interaction, responsive layout, and source-location navigation.

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

## Notes
General feedback: “Can we have the pins in a little 22 box with the notes of the pins, if it's possible?”

