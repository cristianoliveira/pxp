# Purpose

`scripts/` is reserved for repository support tooling, such as maintenance and evaluation helpers.

# Boundaries

Scripts are not runtime CLI capabilities. They may inspect or package repository inputs, but must preserve deterministic behavior and must not become a hidden replacement for [command orchestration](internal/pixelperfectcmd/AGENTS.md).

# Connections

- [Skills](skills/AGENTS.md): owns agent workflow content that support scripts may package or validate.
- [Evaluation controls](tests/evals/AGENTS.md): owns evaluation inputs and evidence.

# Placement

Put reusable support automation here when it serves repository maintenance across modules. Keep domain logic in the owning runtime package.