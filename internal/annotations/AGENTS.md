# Purpose

`internal/annotations` owns the versioned JSON contract for semantic screenshot annotations, coordinate validation, persistence, and region intersections.

# Boundaries

It validates annotation documents and computes geometric matches. It does not own image thresholds, provider protocols, or Cobra workflow policy.

# Connections

- [Command orchestration](internal/pixelperfectcmd/AGENTS.md): loads and dimension-checks annotation documents at the command boundary.
- [Image comparison](internal/imagediff/AGENTS.md): consumes intersection matches to enrich comparison regions.
- [Output](internal/output/AGENTS.md): provides safe file creation used when writing the JSON contract.

# Landmarks

- `internal/annotations/annotations.go:Load`: decodes and validates a versioned document.
- `internal/annotations/annotations.go:Document.Intersections`: returns annotation matches for a comparison region.
- `internal/annotations/annotations.go:Write`: persists a document.

# Boundary flows

- Information flow: `internal/annotations/annotations.go:Load` -> `internal/imagediff/image.go:CompareImagesWithThresholds` via `internal/pixelperfectcmd/command.go:NewCommand`; value: `annotations.Document`.

# Placement

Keep coordinate-space rules and intersection math here. Put image-derived metrics in `imagediff` and command input policy in `pixelperfectcmd`.