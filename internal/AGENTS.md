# Purpose

`internal/` contains the private runtime capabilities behind [the `pxp` executable](cmd/AGENTS.md).

# Boundaries

- [Command orchestration](internal/commands/AGENTS.md) owns options, validation, and workflow sequencing.
- [Image comparison](internal/imagediff/AGENTS.md) owns deterministic PNG evidence.
- [Visual context](internal/imagecontext/AGENTS.md) owns optional provider protocols.
- [Annotations](internal/annotations/AGENTS.md) owns annotation data integrity and geometry.
- [Reports](internal/report/AGENTS.md) owns HTML presentation.
- [Output](internal/output/AGENTS.md) owns structured output and serialization.
- [Artifact persistence](internal/artifact/AGENTS.md) owns filesystem artifact creation and writing.
- [CLI runtime](internal/cli/AGENTS.md) owns shared process helpers and error contracts.

# Connections

The command package coordinates capabilities. Core image evidence may consume annotation geometry and artifact persistence; provider and report packages consume comparison results without changing their meaning.

# Placement

Place a responsibility in the narrowest package that owns its decisions. Keep composition in `commands`, infrastructure adapters at the edges, and deterministic analysis independent of external services.
