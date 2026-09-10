# Purpose

`skills/pxp` guides agents through implementing a UI from a reference screenshot, measuring progress with the standalone CLI, and reporting evidence honestly.

# Boundaries

The skill describes an iterative workflow and references stable CLI contracts. It must not contain the target implementation or leak evaluation answers; holdout controls belong in [evaluation fixtures](../../tests/evals/AGENTS.md).

# Connections

- [pxp command](cmd/AGENTS.md): provides the executable workflow the skill invokes.
- [Image comparison](internal/imagediff/AGENTS.md): provides deterministic measurements.
- [pxp orchestration](internal/pixelperfectcmd/AGENTS.md): defines command options and artifacts.
- [Skill evaluations](skills/pxp/evals/AGENTS.md): validates packaging and workflow guidance.

# Placement

Update this skill when the agent workflow or CLI contract changes. Keep measured reference fixtures and accepted answers outside `skills/` so they cannot be copied into candidate runs.
