---
id: TASK-0014
title: Add implementation context to review rounds
status: done
depends_on: []
priority: high
tags: []
---

# Add implementation context to review rounds

## Problem
A screenshot alone does not tell the human what the agent changed, what to test, or what outcome is expected. Without concise implementation context, review becomes visual guesswork and the human may approve without checking the intended behavior.

## Desired outcome
Before inspecting screenshots, the human can understand the purpose of this round: what changed, what to test, and what outcome the agent expects. Context supports informed review; it does not replace the screenshots or human decision.

## Product shape to evaluate
Provide a concise, structured round brief near the review header, with a clear title and optional sections:
- **What changed:** implementation summary in the agent's words.
- **What to test:** concrete behavior or visual areas the human should exercise.
- **Expected outcome:** what should now be true.
- **Known limitations / not in scope:** prevent false bug reports and unsafe assumptions.
- **Source or change reference:** optional commit, task, or file link when available.

Keep the brief scannable and visibly distinct from human feedback. Do not require a long prose report or expose untrusted raw agent output as trusted fact.

## Acceptance criteria
- [ ] A review round can carry a title and structured implementation context without changing immutable screenshot identity, annotation coordinates, or decision semantics.
- [ ] The UI presents the title and context before or beside the evidence, with a clear reading order and collapse/expand behavior that is keyboard-accessible. The reviewer can still reach Reference/Current/Overlay and decision controls quickly.
- [ ] “What changed” and “What to test” are distinct and concrete. Empty or missing context has a sensible fallback; context never blocks review.
- [ ] Context is available to the agent in the structured round result/feedback provenance and remains linked across rounds without being silently rewritten. Human notes remain distinguishable from agent context.
- [ ] Context is safe to render: escape untrusted text, preserve line breaks/readability, cap lengths, and avoid executable markup or accidental links. Any source/commit references are explicit and non-authoritative.
- [ ] Keyboard and assistive technology users can discover, read, collapse, and expand context; state and headings are programmatically exposed, focus is not lost, and context does not rely on color.
- [ ] A reviewer can use the context to verify an implementation through a realistic round: inspect the stated change, exercise the stated test path, submit feedback or explicitly approve, and see the context preserved in the outcome.
- [ ] Add contract, persistence, rendering, accessibility, and browser regression coverage. Test omitted context, long/multiline text, hostile markup, previous-round linkage, and both Submit and Approve outcomes.

## Discovery questions
- Should context be supplied as CLI flags, a versioned brief file, structured stdin, or an agent skill convention? Prefer a machine-readable contract with a human-readable fallback.
- Is context authored by the agent, the human, or both? Treat agent-authored claims as hypotheses for review, not approval evidence.
- Which fields are required for a useful round? Start with title plus What changed and What to test; keep limitations and source reference optional.
- Should context be copied into every round or linked to a prior brief? Preserve the exact context used for each immutable round.

## Scope and relationships
Planning only; no runtime authorization in this task. This is distinct from TASK-0011 transport/foreground handoff, TASK-0010 decision confirmation, TASK-0013 popup discovery, and the existing Focus visual baseline. Preserve the product vision in `docs/review-loop-vision.md`: context should make review calmer, more accessible, and more informed—not add clutter or substitute for human judgment.

