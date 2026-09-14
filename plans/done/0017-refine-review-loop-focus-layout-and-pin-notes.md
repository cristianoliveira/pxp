---
id: TASK-0017
title: Remove redundant Focus-mode framing from review loop
status: done
depends_on: []
priority: normal
tags: []
---

# Remove redundant Focus-mode framing from review loop

## Problem
The review page repeats Focus-mode framing even though the page is always in Focus mode, adding hierarchy that reviewers do not need.

## Context
Feedback from review session `review-53ae0a1ed5a45014-d39e11712cc3c70c`.

## Acceptance criteria
- [x] Remove the redundant Focus-mode framing identified in the review annotation.
- [x] Preserve clear evidence and feedback hierarchy.
- [x] Preserve keyboard access, annotation editing, draft persistence, and distinct Submit feedback / Approve actions.
- [x] Add browser coverage for the simplified Focus layout.

## Implementation slice

Removed the always-present `Focus mode` heading and explanatory subtitle from
`internal/review/frontend/index.html`, while retaining the `pxp / human review`
context label and agent-waiting status badge. Evidence, annotation controls,
feedback, draft, and distinct decision actions remain unchanged.

Added `checkSimplifiedFocusLayout` to the browser harness. It asserts that the
redundant framing is absent and the evidence, feedback, Submit, Approve, and
agent-waiting affordances remain visible. The existing browser suite continues
to cover keyboard access, annotation creation/edit/removal, draft persistence,
and decision confirmation.

Visual proof: `/tmp/task0017-browser-proof/task0017-focus-layout.png`.

## Notes
Reference annotation: `reference` rectangle `(18,47,367,69)`: “We don't need this basically; Since it's always focus mode”.

