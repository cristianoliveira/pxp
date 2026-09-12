---
id: TASK-0006
title: POC local annotated visual review loop
status: done
depends_on: []
priority: normal
tags: []
---

# POC local annotated visual review loop

## Problem
Users need to point to screenshot differences and send location-aware feedback to an agent across successive review rounds until approval.

## Context
User authorized a POC, then explicitly required a review loop. Scope: compare -> human feedback -> agent fixes -> new comparison -> repeat until human approves. A fresh short-lived server per round is sufficient; no autonomous code-editing engine is required.

## Acceptance criteria
- [ ] Localhost-only temporary server displays before, after, and overlay for a comparison.
- [ ] Human can add general notes, image pins, and rectangle notes.
- [ ] Structured feedback includes stable note IDs, image identity, and original-pixel coordinates unaffected by display scaling.
- [ ] Agent remains blocked until an explicit Submit feedback or Approve action. Submit requires at least one note; Approve ends the loop without notes. CLI exposes the decision and saved feedback path for an agent.
- [ ] Closing the browser preserves the pending round, which reopens at the same URL; explicit server termination or an opt-in expiry returns a failure, never an empty decision.
- [ ] Feedback references immutable comparison inputs; a subsequent round can link to prior feedback without overwriting it.
- [ ] Notes persist while reviewing and remain available after server shutdown.
- [ ] Validation, lifecycle, persistence, and coordinate behavior have deterministic tests; browser interaction is independently checked where tooling permits.
- [ ] Documentation gives a runnable multi-round example and identifies POC limitations.

## Notes
Dave owns implementation and focused tests. Kelly owns independent acceptance verification. Mony owns coordination and task closeout. Exclude accounts, collaboration, freehand drawing, and live edits during an active review round.

