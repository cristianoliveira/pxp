# Purpose

`skills/pxp` guides an agent through implementing or refining a real UI from a reference screenshot, using `pxp` for deterministic measurement and reporting evidence honestly.

# Boundaries

The skill owns workflow guidance and stable command usage. It does not own target application code, screenshot capture tooling, deterministic metrics, or holdout evaluation answers.

# Connections

- [Executable commands](cmd/AGENTS.md): provide the CLI workflow invoked by the skill.
- [Command orchestration](internal/commands/AGENTS.md): defines command options and artifact sequencing.
- [Image comparison](internal/imagediff/AGENTS.md): provides measurements and advisory image evidence.
- [Evaluation packaging](skills/pxp/evals/AGENTS.md): checks skill metadata and fixture packaging.
- [Evaluation controls](tests/evals/AGENTS.md): owns independent holdout inputs and accepted evidence.

# Placement

Update this skill when the agent workflow or public CLI contract changes. Keep implementation examples and accepted solutions outside the skill so the procedure cannot leak evaluation answers.