# Purpose

`internal/annotations` defines the versioned JSON contract for semantic screenshot annotations, validates coordinate spaces, and computes annotation/region intersections.

# Boundaries

It owns annotation data integrity and intersection math. It does not know external integrations, image comparison thresholds, or Cobra command policy.

# Connections

- [Image comparison](internal/imagediff/AGENTS.md): consumes intersection matches to enrich mismatch regions.
- [Output](internal/output/AGENTS.md): supplies filesystem helpers for annotation files.
- [pxp orchestration](internal/pixelperfectcmd/AGENTS.md): loads annotation documents at the command boundary.

# Landmarks

- `internal/annotations/annotations.go:Load`: validates a versioned annotation document.
- `internal/annotations/annotations.go:Write`: persists the annotation contract.
- `internal/annotations/annotations.go:Document.Intersections`: maps a region to matching annotations.

# Boundary flows

- Information flow: `internal/annotations/annotations.go:Load` -> `internal/imagediff/image.go:CompareImagesWithThresholds` via `internal/pixelperfectcmd/command.go:NewCommand`; value: `annotations.Document`.

# Placement

Keep coordinate validation and intersection math here. Put image metrics in `imagediff` and input/flag handling in `pixelperfectcmd`.
