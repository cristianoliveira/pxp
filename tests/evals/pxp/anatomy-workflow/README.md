# Anatomy workflow eval controls

This directory contains immutable offline controls for TASK-0025. Prompts are
agent-facing and intentionally do not contain accepted geometry. Expected
geometry and error assertions live separately under `expected/`.

Cases:

- `simple-control`: exact bounds, pixels, and colors.
- `grouped-control`: raw separated regions merged with `--group 4`.
- `empty-control`: valid solid image with an explicit empty result.
- `truncated-control`: bounded output and exact `--full` recovery hint.
- `malformed-control`: invalid PNG and operational recovery.
- `ambiguous-layout-holdout`: screenshot-like layout with geometry anchors only;
  the rubric forbids semantic UI labels.

Fixture SHA-256 values are enforced by
`skills/pxp/evals/test_anatomy_workflow_contract.py`. Do not replace a fixture
without changing the manifest and recording why the control changed.

The deterministic harness compares one bounded anatomy invocation with setup
controls for comparison, probe, scan, and review. Those controls measure bytes,
errors, and command availability; they do not pretend that comparison or a
single probe provides element geometry. Review is not started because it needs
an interactive browser decision.
