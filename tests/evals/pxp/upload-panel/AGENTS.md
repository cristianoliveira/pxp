# Purpose

This directory is the holdout control set for the upload-panel `pxp` evaluation. It owns accepted evidence, deliberate defects, alternatives, and coordinator checks.

# Boundaries

The controls are grader inputs, not skill content or production examples. Verification must inspect the extracted application and browser behavior rather than trust claims or filenames.

# Connections

- [Evaluation root](tests/evals/AGENTS.md): owns evaluation scope and placement policy.
- [The `pxp` skill](skills/pxp/AGENTS.md): is evaluated against these controls.

# Placement

Keep accepted snapshots and control metadata immutable by default. Add a control only when it isolates one defect or documented alternative without leaking the solution into the skill.