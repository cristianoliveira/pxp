# Purpose

`internal/output` owns the stable output boundary: TOON-first structured rendering, compatibility JSON, and raw text/file modes.

# Boundaries

Callers provide domain values and choose a format. This package does not know image-analysis policy, command semantics, provider configuration, or report presentation.

# Connections

- [CLI runtime](../cli/AGENTS.md): selects and binds the printer to command streams.
- [Command orchestration](../commands/AGENTS.md): supplies comparison and diagnostic values for rendering.
- [Annotations](../annotations/AGENTS.md) and [reports](../report/AGENTS.md): consume [artifact](../artifact/AGENTS.md) for filesystem persistence.

# Landmarks

- `internal/output/printer.go:New`: constructs a format-specific printer.
- `internal/output/printer.go:Printer.Structured`: emits TOON or compatibility JSON.
- `internal/output/printer.go:Printer.File`: emits raw or structured file-mode output.

# Boundary flows

- Information flow: `internal/imageio/imageio.go:CompareImagesWithThresholds` -> `internal/output/printer.go:Printer.Structured` via `internal/commands/command.go:NewCommand`; value: `imagediff.ImageComparison`.

# Placement

Add shared envelopes or renderers here only when multiple producers need the same output policy. Keep domain-specific result construction with its producer.