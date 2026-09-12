# Review loop: product vision and goals

## Vision
Make visual review a calm, accessible conversation between a human and an agent.

The human can inspect evidence, explain what should change, and return when ready. The agent receives precise feedback, makes changes, and presents another round. Neither side needs to guess what happened or what comes next.

This document defines desired outcomes, not a frontend implementation recipe. It records the product direction agreed with the user; it does not claim the current application meets every goal.

## Goals

### 1. The entire review works from the keyboard
A keyboard-heavy user can inspect all views, create and revise pins and rectangles, navigate feedback, and make a decision without reaching for a mouse.

Success means a keyboard-only journey has no missing operation, hidden focus, trap, or unexpected loss of work. Helpful shortcuts are discoverable, but ordinary keyboard navigation remains sufficient.

### 2. Review is accessible, not merely operable
People can understand controls, evidence context, errors, and progress without depending on color, pointer precision, or visual status alone.

Success means the review controls target WCAG 2.2 AA, retain usability under zoom, and expose meaningful names, state, feedback geometry, and status to assistive technology. Keyboard and screen-reader evidence accompanies delivery; unverified areas are named rather than presented as conformance. Image descriptions must not invent visual findings.

### 3. The interface is comfortable to look at and easy to navigate
Evidence gets the most space. Controls and feedback are easy to find. Switching views does not make the user reorient or hunt for the same detail.

Success means Reference, Current, and Overlay occupy a stable primary image area, the active view is clear, and feedback stays easy to reach. Readable type, restrained visual noise, accessible contrast, and consistent spacing support extended review.

The selected visual baseline is Focus mock selected during product review (`docs/review-mocks/focus.html` in the separate review implementation work), including its overall layout, large image area, view controls, feedback sidebar, spacing, and styling—not just its ability to switch images. Accessibility and real annotation behavior improve that baseline. Replacing the visual direction requires an explicit product decision, not an incidental implementation shortcut.

### 4. Feedback keeps its meaning across the human–agent boundary
The agent can identify what the human pointed to and which evidence they were viewing without reconstructing context from chat.

Success means every spatial note retains its round, source view, original-image geometry, and note text. Selecting a note returns the human to its evidence. General feedback remains possible. Earlier feedback stays attached to its original round, not silently transferred to a new screenshot.

### 5. The human controls the pace and decision
The agent waits while the human reviews. Leaving the browser is not a decision. The application never treats silence as consent.

Success means Submit feedback explicitly hands work to the agent, Approve explicitly ends the review, and empty feedback cannot masquerade as either. Pending review can be reopened. Operational failure is reported as failure, not approval. Waiting, submitted, approved, and failed states are distinguishable.

### 6. Each round makes progress understandable
The human knows what is being reviewed, what the agent changed, what remains unresolved, and what action comes next.

Success means subsequent rounds preserve the feedback trail and offer a concise account of changes and unresolved requests. Agent claims of addressing feedback remain distinct from human acceptance. A full history dashboard is not required to achieve this outcome.

## How we move toward these goals

Each change starts with the user outcome it improves and the goals it must preserve. A small, working slice is preferred to a broad redesign that cannot be reviewed.

Before implementation, name its observable acceptance conditions and visual baseline. During delivery, compare the running page with that baseline, exercise the complete keyboard journey, and inspect the agent-facing feedback—not only the UI controls or automated test results.

At handoff, show the actual application, state which goals have evidence, and name remaining gaps. Do not close work merely because a feature exists or a test suite is green.

Two decisions must stay separate:

- **Round approval:** the human approves the screenshot evidence being reviewed.
- **Product acceptance:** the agreed review application experience has been delivered and verified, including its visual direction and accessibility requirements.

A saved `decision: approved` proves the first, not the second. A mock, technical QA result, or approval of unrelated image fixtures does not establish product acceptance.

Keep this document stable across tasks. Tasks link to these goals and record their own evidence. Change the vision only when the product direction changes explicitly; do not rewrite goals to fit an incomplete implementation.
