# Purpose

`internal/` contains the private runtime capabilities behind [the `pxp` executable](cmd/AGENTS.md).

# Boundaries

- [Command orchestration](internal/commands/AGENTS.md) owns options, validation, and workflow sequencing.
- [Image comparison](internal/imagediff/AGENTS.md) owns deterministic image evidence.
- [Image I/O](internal/imageio/AGENTS.md) owns PNG/file adapters for image comparison.
- [Annotation persistence](internal/annotationio/AGENTS.md) owns annotation JSON/file adapters.
- [Artifact persistence](internal/artifact/AGENTS.md) owns generic filesystem creation.
- [Visual context](internal/imagecontext/AGENTS.md) owns optional provider protocols.
- [Annotations](internal/annotations/AGENTS.md) owns annotation data integrity and geometry.
- [Review](internal/review/AGENTS.md) owns localhost annotated review rounds.
- [Reports](internal/report/AGENTS.md) owns HTML presentation.
- [Output](internal/output/AGENTS.md) owns structured output and serialization.
- [CLI runtime](internal/cli/AGENTS.md) owns shared process helpers and error contracts.

# Connections

The command package coordinates capabilities. Core image evidence consumes in-memory inputs; image and annotation persistence adapters sit at the edge. Provider and report packages consume comparison results without changing their meaning.

# Placement

Place a responsibility in the narrowest package that owns its decisions. Keep composition in `commands`, infrastructure adapters at the edges, and deterministic analysis independent of external services.
