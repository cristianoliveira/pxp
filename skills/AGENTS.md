# Purpose

`skills/` contains agent-facing workflows for using `pxp`, separate from production runtime code.

# Boundaries

- [The `pxp` skill](skills/pxp/AGENTS.md) guides screenshot-driven UI implementation and measured refinement.
- `pxp-review-loop` guides human-annotated review rounds and explicit approval handoff.
- Skill evaluation packaging belongs to [the evaluation helpers](skills/pxp/evals/AGENTS.md).

Skill content may explain stable CLI contracts but must not implement application code or contain holdout answers.

# Connections

- [CLI commands](cmd/AGENTS.md): provide the executable invoked by the workflow.
- [Image comparison](internal/imagediff/AGENTS.md): provides deterministic measurements.
- [Evaluation controls](tests/evals/AGENTS.md): independently holds grader inputs and accepted evidence.

# Placement

Put reusable agent procedure in `skills/`. Keep reference fixtures, accepted implementations, and evaluation controls outside the skill content.