---
id: TASK-0011
title: Resume agent directly from blocking review completion
status: todo
depends_on: []
priority: high
tags: []
---

# Resume agent directly from blocking review completion

## Problem
The human currently has to return to chat to announce that browser feedback was submitted. A detached review server without an awaited completion handoff breaks the intended human-agent loop: the decision must wake the waiting agent automatically and end the round server lifecycle.

## Desired outcome
Agent starts review → human receives URL → agent waits → human confirms Submit → feedback is persisted → CLI returns the structured outcome and stops its server → agent reads the feedback and continues without another human chat message. On Approve, the agent receives that explicit outcome and ends the loop instead of making another change.

## Existing evidence and scope
`internal/commands/review.go` already starts a server, prints its URL to stderr, waits through `server.WaitContext(cmd.Context())`, emits structured completion, and defers server closure. The installed workflow in `skills/pxp-review-loop/SKILL.md` already describes waiting, but redirects both streams and tells the agent to read the URL afterward. A non-streaming tool can make that sequence impossible until the command completes, encouraging detached execution.

First reproduce the missing agent handoff in the actual harness. This may be a workflow/tool-execution integration gap rather than missing CLI blocking. Do not build a second wait mechanism or a new daemon without evidence.

## Acceptance criteria
- [ ] The human receives a usable localhost URL while the review command is still running; URL discovery cannot depend on waiting for the final result.
- [ ] Once review starts, the initiating agent remains in an awaited review operation. It does not finish its turn with instructions to return to chat, start unrelated implementation, or rely on repeated status polling to notice the decision.
- [ ] Valid explicit Submit causes the command's completion to resume the agent with a structured decision and feedback path, without the human sending any further chat message. The agent then reads the saved feedback before acting.
- [ ] Approve returns its actual structured outcome to the waiting agent and ends this review loop. Neither absence of notes nor process exit alone implies approval.
- [ ] The round-owned server closes after successful decision handoff; no leftover listener or detached review process is needed. Use graceful lifecycle cleanup, not an unrelated-process kill. Persisted screenshots and feedback remain available after shutdown.
- [ ] Invalid/empty submission and canceled confirmation leave the command waiting. Browser close/reopen before a decision remains recoverable at the same URL. Explicit cancellation/server failure returns an operational failure, never a successful empty result.
- [ ] Host tool time limits do not silently kill an expected human wait or require another human message to resume. Document the supported waiting/streaming mechanism and explicit timeout behavior; do not assume this harness supports indefinite synchronous tool calls.
- [ ] Preserve immutable round identity, previous-feedback linkage, and one final outcome. No duplicate agent continuation or silent reclassification after retries.
- [ ] Demonstrate a real two-round run tied to editable source: human submits notes, agent receives them and makes the requested source change, recaptures, starts next review, and waits again. A fixture-only demo can test transport but cannot prove an implementation loop.

## Delivery plan
1. Lead/dev inspect actual tool capabilities: live URL delivery while blocked, completion notification, cancellation, and time limits. Reproduce current failure and identify the narrowest boundary that needs change.
2. Prefer existing foreground CLI wait with supported tool streaming. If launch and await must be separate tool operations, they must form one owned workflow: deliver URL then immediately await completion. No orphaned server with a final chat response asking the human to notify the agent.
3. Update workflow instructions and only the runtime/integration behavior shown to be missing. Keep persistence and server lifecycle under the existing owner.
4. QA records URL delivery, pending operation, submitted payload, agent continuation, process/listener cleanup, failure paths, and next-round start. Never fabricate a user submission or approval.

## Relationships and non-goals
TASK-0010 covers accidental decision confirmation; this task preserves its intent but does not depend on implementation starting there. TASK-0008 visual/product acceptance is separate. No persistent daemon, new collaboration service, autonomous editing inside pxp, or changes to image metrics. Plan only until lead assigns implementation.

