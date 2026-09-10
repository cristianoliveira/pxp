# Purpose

This directory is the holdout control set for the upload-panel pxp evaluation. It contains accepted evidence, deliberate defects, alternatives, and coordinator checks.

# Boundaries

The controls are grader inputs, not skill content or production examples. Verification must exercise the extracted application and browser behavior rather than trust candidate claims or filenames.

# Connections

- [Evaluation root](../../AGENTS.md): owns the evaluation scope and placement policy.
- [pxp skill](../../../../skills/pxp/AGENTS.md): is evaluated against these controls.

# Placement

Keep new accepted snapshots and control metadata immutable by default. Add a control only when it isolates one defect or a documented alternative without leaking the solution into the skill.
