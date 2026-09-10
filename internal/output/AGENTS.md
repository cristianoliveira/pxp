# Purpose

`internal/output` owns stable result envelopes, TOON/JSON rendering, text/file compatibility behavior, and safe filesystem artifact creation.

# Boundaries

Callers provide domain values and select the output mode. This package must not know external command semantics, image-analysis policy, or environment configuration.

# Connections

- [CLI wiring](internal/cli/AGENTS.md): selects the format and binds the printer to a command stream.
- [Commands](cmd/AGENTS.md): consume the stable renderer for user-facing results.
- [Internal capabilities](internal/AGENTS.md): provide query/detail values and artifact paths.

# Landmarks

- `internal/output/contracts.go:NewQuery`: creates a non-null collection envelope.
- `internal/output/contracts.go:NewLimitedQuery`: adds bounded-result metadata.
- `internal/output/printer.go:Printer.Structured`: emits TOON by default or compatibility JSON.
- `internal/output/file.go:WriteFile`: writes deterministic artifacts.

# Boundary flows

- Information flow: `internal/extract/inspect.go:InspectTree` -> `internal/output/printer.go:Printer.Structured` via `cmd/root.go:Execute`; value: `extract.InspectOutput`.
- Information flow: `internal/imagediff/image.go:CompareImagesWithThresholds` -> `internal/output/printer.go:Printer.Structured` via `internal/pixelperfectcmd/command.go:NewCommand`; value: `imagediff.ImageComparison`.

# Placement

Add a renderer or envelope here only when it is shared output policy. Keep domain-specific result construction in the producer package.
