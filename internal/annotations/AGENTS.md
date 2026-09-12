# Purpose

`internal/annotations` owns the semantic screenshot annotation domain: coordinate validation and region intersections. JSON/file persistence lives in [annotationio](internal/annotationio/AGENTS.md).

# Boundaries

It validates annotation documents and computes geometric matches. It does not own image thresholds, provider protocols, or Cobra workflow policy.

# Connections

- [Command orchestration](internal/commands/AGENTS.md): loads and dimension-checks annotation documents at the command boundary.
- [Image comparison](internal/imagediff/AGENTS.md): consumes intersection matches to enrich comparison regions.
- [Annotation persistence](internal/annotationio/AGENTS.md): decodes and serializes the JSON contract at the adapter boundary.

# Landmarks

- `internal/annotationio/annotationio.go:Load`: decodes JSON and delegates domain validation.
- `internal/annotations/annotations.go:Document.Intersections`: returns annotation matches for a comparison region.
- `internal/annotations/validation.go:Document.Validate`: checks domain invariants without I/O.

# Boundary flows

- Information flow: `internal/annotationio/annotationio.go:Load` -> `internal/annotations/annotations.go:Document.Intersections` via `internal/commands/command.go:NewCommand`; value: `annotations.Document`.

# Placement

Keep coordinate-space rules and intersection math here. Put image-derived metrics in `imagediff` and command input policy in `commands`.