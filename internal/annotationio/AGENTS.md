# Purpose

`internal/annotationio` owns the JSON and filesystem adapter for [annotation documents](internal/annotations/AGENTS.md).

# Boundaries

It translates the wire DTO into the annotation domain and persists validated documents. It does not own coordinate rules, image comparison, or command policy.

# Connections

- [Annotations](internal/annotations/AGENTS.md): provides the domain document that this adapter validates and serializes.
- [Artifact persistence](internal/artifact/AGENTS.md): provides parent-directory creation and file creation.
- [Command orchestration](internal/commands/AGENTS.md): consumes loaded documents at the comparison boundary.

# Landmarks

- `internal/annotationio/annotationio.go:Load`: decodes, rejects trailing JSON, and validates an annotation document.
- `internal/annotationio/annotationio.go:Write`: serializes an annotation document to JSON.

# Boundary flows

- Information flow: `internal/annotationio/annotationio.go:Load` -> `internal/annotations/annotations.go:Document.Intersections` via `internal/commands/command.go:NewCommand`; value: `annotations.Document`.

# Placement

Keep wire-format translation and file access here. Put annotation invariants in `annotations` and generic filesystem creation in `artifact`.
