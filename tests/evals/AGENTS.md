# Purpose

`tests/evals/` owns offline evaluation controls and retained evidence for the [agent-facing `pxp` workflow](skills/pxp/AGENTS.md).

# Boundaries

This tree contains holdout inputs, accepted or alternative evidence, and independent verification helpers. It is not production code and must not be treated as skill content or a target implementation.

# Connections

- [The `pxp` skill](skills/pxp/AGENTS.md): is evaluated against these controls.
- [Evaluation packaging](skills/pxp/evals/AGENTS.md): prepares and validates skill-side inputs.

# Placement

Keep new controls immutable by default and add one only when it isolates a distinct defect or documented alternative without revealing the target solution.