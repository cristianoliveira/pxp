# Purpose

`docs/` contains user-facing contracts for the `pxp` CLI, including command behavior, output formats, metrics, and examples.

# Boundaries

Documentation describes the stable interface exposed by [the executable](cmd/AGENTS.md) and [command orchestration](internal/pixelperfectcmd/AGENTS.md). It does not define runtime behavior or evaluation answers.

# Connections

- [Command orchestration](internal/pixelperfectcmd/AGENTS.md): is the source of command policy and options.
- [Output](internal/output/AGENTS.md): defines structured output behavior documented here.
- [Skill workflow](skills/pxp/AGENTS.md): consumes the documented CLI contract.

# Placement

Put durable user-facing contracts here. Keep implementation ownership in the nearest runtime guide and update examples when the public command contract changes.