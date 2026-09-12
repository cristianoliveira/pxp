---
id: TASK-0007
title: Focus review UX with view-bound annotations
status: todo
depends_on: []
priority: high
tags: []
---

# Focus review UX with view-bound annotations

## Problem
Reviewers need to compare reference, current, and overlay in the same space and annotate the evidence they are viewing. Feedback loses meaning when the agent cannot distinguish a reference target from a current defect or overlay mismatch.

## Desired outcome
The human reviews one large image, switches between Reference / Current / Overlay, and leaves feedback tied to the exact view and round. The agent receives enough context to interpret each note without guessing which image it concerns.

## Context and authorization
- User selected `docs/review-mocks/focus.html` over the desk and round-history concepts.
- User explicitly requested pins on Overlay and Reference as well as Current, with reference context conveyed to the agent.
- User authorized team implementation. The HTML mock is a direction, not production code or a complete behavior specification.
- Builds on completed TASK-0006. No unfinished dependency blocks start.

## Acceptance criteria
- [ ] One primary image area switches between clearly labeled Reference / Current / Overlay views without changing image alignment or displayed scale merely because the view changes.
- [ ] Active view is visually clear and keyboard-accessible. Existing overlay evidence remains available; do not replace it with the mock's approximate alpha blend.
- [ ] Reviewers can create pins on all three views. Apply the same source-view behavior to existing rectangle annotations; do not regress rectangle support.
- [ ] Each annotation persists its source view, original-image pixel geometry, and round association. Agent-readable feedback distinguishes reference targets, current observations, and overlay observations without inferring intent from the view alone.
- [ ] Existing wire naming such as `actual` can remain; UI may say Current. Mapping is explicit and existing persisted feedback remains interpretable without silent relabeling.
- [ ] Feedback list visibly labels each annotation's source view. Selecting a note opens that view and highlights its annotation. By default, other-view annotations do not appear as though they belong to the active view.
- [ ] View switching preserves draft notes and geometry. Closing/reopening the browser restores the pending draft, including annotation source views; it never submits or approves.
- [ ] Agent remains blocked until explicit valid Submit feedback or Approve. Empty feedback is rejected; approval does not silently discard draft feedback. Service termination remains an explicit operational failure.
- [ ] Submitted feedback is persisted against the immutable round. A subsequent review retains previous-feedback linkage and does not move earlier annotations onto new screenshots.
- [ ] UI makes agent-waiting state and distinct Submit feedback / Approve outcomes clear. General notes remain supported.

## Delivery and ownership
1. Mony (lead): confirm implementation ownership and integration scope; keep this task open until evidence satisfies acceptance.
2. Dave (dev): inspect current contracts; add failing tests for view-bound feedback and failure paths, then implement Focus layout and annotation interactions. Preserve runtime architecture and existing CLI contracts.
3. Kelly (QA): independently test all three views, scaled pins and rectangles, note-to-view navigation, draft reopen, invalid decisions, and a two-round handoff. Coordinate a real human submission; do not fabricate the user's approval.
4. Mony: integrate, verify available watcher gate/freshness or record fallback evidence, and provide runnable user-review URL before closing task.

## Verification boundaries
Use behavioral tests for payload persistence and validation, and browser checks for view switching and geometry. Include reference/current/overlay points and rectangles at non-native display size; verify source view plus coordinates in saved feedback. Measure affected coverage/regression risk. Do not substitute markup-string assertions for interaction evidence.

## Non-goals
No authentication, collaboration, freehand tools, new zoom/pan system, full history dashboard, autonomous edits inside the review service, or changes to comparison metrics. No timeout-default change in this task. Prototype files need not become runtime assets.

