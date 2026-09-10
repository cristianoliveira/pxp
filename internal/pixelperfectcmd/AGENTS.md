# Purpose

`internal/pixelperfectcmd` composes the standalone image-comparison command: option validation, input crops and metadata, deterministic comparison, bounded diagnostics, optional annotations/visual context, and artifact orchestration.

# Boundaries

This package owns Cobra workflow policy but not image algorithms, provider protocol details, or HTML template ownership. It is used by the standalone [pxp executable](cmd/AGENTS.md).

# Connections

- [Image comparison](internal/imagediff/AGENTS.md): performs deterministic metrics and artifacts.
- [Visual context](internal/imagecontext/AGENTS.md): provides optional advisory descriptions.
- [Annotations](internal/annotations/AGENTS.md): loads and validates semantic region metadata.
- [Reports](internal/pixelperfectreport/AGENTS.md): renders HTML reports from comparison results.
- [CLI wiring](internal/cli/AGENTS.md): supplies output, errors, and exit-code behavior.
- [Output](internal/output/AGENTS.md): renders structured command results.

# Landmarks

- `internal/pixelperfectcmd/command.go:NewCommand`: creates the standalone Cobra command.

# Boundary flows

- Information flow: `internal/imagediff/image.go:CompareImagesWithThresholds` -> `internal/pixelperfectreport/report.go:Render` via `internal/pixelperfectcmd/command.go:NewCommand`; value: `imagediff.ImageComparison`.
- Information flow: `internal/imagediff/image.go:CompareImagesWithThresholds` -> `internal/output/printer.go:Printer.Structured` via `internal/pixelperfectcmd/command.go:NewCommand`; value: `imagediff.ImageComparison`.

# Placement

Put command option and orchestration policy here. Put reusable image evidence in `imagediff`, provider adapters in `imagecontext`, and artifact templates in `pixelperfectreport`.
