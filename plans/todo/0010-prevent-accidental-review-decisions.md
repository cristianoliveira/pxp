---
id: TASK-0010
title: Prevent accidental review decisions
status: todo
depends_on: []
priority: high
tags: []
---

# Prevent accidental review decisions

## Problem
The user reports that round-8 Submit feedback was a misclick rather than the intended decision. Adjacent final actions can prematurely hand work to the agent or end review; the UI needs an accessible opportunity to verify intent before persisting an irreversible outcome.

## Context
Planning only: no runtime change is authorized by this task's creation. Preserve the goals in `docs/review-loop-vision.md`, especially keyboard completeness and human control.

This is distinct from TASK-0008 visual UI acceptance. TASK-0008 later received a separate, persisted explicit Approve decision; Round-8 Submit remains historical feedback and must not be relabeled. A screenshot-round decision alone still does not replace verification of visual and accessibility criteria.

## Decision alternatives to evaluate before implementation
1. Recommended: a shared, explicit review step after choosing either action. Show the selected outcome, round, note count, and consequence; final buttons name the actual action (for example, Send feedback to agent / Approve and finish), never generic OK. Back returns to the intact draft.
2. Alternative: inline confirmation next to the selected action, with equally explicit outcome and cancel behavior. This avoids a modal but must remain obvious and reachable at zoom and through screen readers.
3. Supporting improvement for either option: separate and visually distinguish Submit feedback from Approve with clear consequence text. Color or spacing alone is not sufficient protection from an accidental final decision.

Select one confirmation pattern after a keyboard and pointer walkthrough. Do not implement all alternatives or add typing challenges, timers, or hold-to-confirm barriers by default.

## Acceptance criteria
- [ ] Choosing Submit or Approve first opens a review/confirmation state, not a persisted final decision or agent wakeup. Final confirmation clearly identifies the chosen outcome and round.
- [ ] Submit feedback communicates that notes will be handed to the agent for another round; Approve communicates that review ends. Neither is a generic primary action whose meaning silently changes.
- [ ] The confirmation step summarizes relevant feedback and consequences. Empty submission remains invalid. Approval never silently discards pending feedback; preserve existing draft-conflict validation.
- [ ] Back/Cancel and Escape dismiss confirmation without committing, clearing notes, or waking the agent. Browser close/reopen from an unconfirmed state restores the pending draft, never confirms it automatically.
- [ ] Both decision paths are keyboard-complete. Enter in a note field creates text as appropriate, not a final decision. Opening confirmation must not let the same keypress, key repeat, double click, or click-through also confirm it. No global submit/approve shortcut bypasses confirmation.
- [ ] Focus is visible and deterministic on entry and exit. Cancellation restores the invoking control. Confirmation has a programmatically exposed name, selected outcome, and consequence. If modal, follow accessible dialog focus containment and restoration; if inline, announce and expose the review region without trapping focus.
- [ ] Agent remains blocked until an explicit valid final confirmation has been successfully persisted. Cancel, silence, expired wait, and server failure are not approval or successful submission.
- [ ] Final feedback is written once with the exact confirmed structured decision and immutable round association. Duplicate/retried requests do not create competing outcomes. Failure before persistence is reported honestly; a lost response after persistence is recoverable without converting or overwriting the saved decision.
- [ ] An already persisted outcome is not silently undone or reclassified. If another review is needed after an accidental committed decision, start an explicit new round with provenance; retain the earlier evidence.

## Verification plan
- Test both decisions from initial selection through cancellation and final confirmation; assert saved payload and agent-unblocking behavior, not only button appearance.
- Test empty Submit, Approve with draft feedback, stale/duplicate requests, persistence failure, and response-loss recovery.
- Browser-check keyboard and pointer use: Enter/Space, Escape/Back, held keys/double activation, focus restoration, zoom, accessible names/status, and note-field typing.
- Close and reopen before confirmation: draft remains pending. Reopen after a committed decision: show its true outcome without allowing a second decision.
- Record product walkthrough evidence for the selected alternative. Do not treat synthetic QA approval as the user's approval.

## Scope and sequencing
No changes to image comparison, annotation geometry, or the Focus visual direction. TASK-0008 is related but does not block planning; no dependency is asserted. Keep this task todo until the lead authorizes implementation after choosing a confirmation pattern.

