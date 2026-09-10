# Purpose

`skills/pxp/evals` packages offline checks for skill metadata, fixture integrity, deterministic fixture packing, and evaluation protocol inputs.

# Boundaries

These helpers validate packaging and evaluation inputs. They do not score model quality, implement target UIs, or replace independent browser and metric review.

# Connections

- [The `pxp` skill](skills/pxp/AGENTS.md): is the subject whose contract and fixtures are checked.
- [Evaluation controls](tests/evals/AGENTS.md): owns holdout implementations, controls, and accepted evidence.

# Placement

Keep schema and packaging checks beside skill inputs. Keep accepted implementations and grader-owned evidence in `tests/evals/`.