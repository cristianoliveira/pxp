---
id: TASK-0008
title: Match runtime Focus UI to approved prototype
status: doing
depends_on: []
priority: high
tags: []
---

# Match runtime Focus UI to approved prototype

## Problem
The user approved the Focus prototype but reports the runtime review page does not visually match it. Functional view switching is insufficient; runtime layout and styling must reproduce the approved Focus direction while preserving annotation and keyboard behavior.

## Context
Approved visual reference: `docs/review-mocks/focus.html` with its shared `style.css` and `app.js`. Current runtime embeds a functional Focus interaction but visibly diverges: it is a plain vertical page with a bordered canvas, while the approved direction uses a light application shell, white header, navigation, card containers, a centered large evidence stage, and a 300px feedback/action sidebar. The runtime must remain an accessible, functional implementation—not an embedded mock or screenshot.

## Acceptance criteria
- [ ] Runtime Focus page matches the approved Focus hierarchy at desktop size: application shell/header, navigation, main evidence card, clearly grouped toolbar/view controls, centered large image stage, and a right feedback/action sidebar.
- [ ] Use the prototype's visual language—light gray page ground, white cards, subtle borders, rounded corners, type hierarchy, spacing, focus treatment, active controls, and primary Submit action—without replacing the runtime overlay evidence with an approximate CSS blend.
- [ ] Preserve the TASK-0007 keyboard-only annotation/edit/remove behavior, semantic controls/status, source-bound geometry, drafts, and explicit Submit/Approve semantics.
- [ ] At narrow widths and 200% zoom, the layout remains usable, maintains reading/action order, and does not hide keyboard or feedback controls.
- [ ] Capture fixed viewport screenshots of prototype and runtime using the same fixture/state. Compare with pxp; document baseline and final metrics/artifacts. Do not resize/crop away layout differences.
- [ ] Browser QA verifies visual hierarchy against the prototype and regression-tests all three views, keyboard annotation paths, draft reopen, invalid decisions, and immutable/two-round behavior.
- [ ] Human approval of a comparison screenshot never closes this task; close only after the user accepts the runtime review UI itself.

## Notes
Dave owns runtime styling/layout parity and visual capture evidence. Kelly owns independent visual/keyboard regression QA. Mony owns live runtime UI review and closure. No mock files become runtime code by copying wholesale; preserve runtime contracts.

