# Purpose

`skills/pixel-perfect/evals` packages offline checks for skill metadata, fixture integrity, deterministic packing, and evaluation protocol.

# Boundaries

These helpers validate evaluation inputs and packaging. They do not score model quality, implement the target UI, or replace independent browser and metric review.

# Connections

- [Pixel-perfect skill](../AGENTS.md): is the subject of the package checks.
- [Evaluation fixtures](tests/evals/AGENTS.md): owns holdout implementation controls and accepted evidence.

# Placement

Keep schema and packaging checks beside the skill inputs. Keep accepted implementations, defect controls, and coordinator-owned evidence in the separate `tests/evals/` tree.
