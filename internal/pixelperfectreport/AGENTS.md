# Purpose

`internal/pixelperfectreport` renders and writes self-contained HTML reports for image-comparison evidence, including provenance, metrics, regions, and embedded PNG artifacts.

# Boundaries

It owns presentation and safe embedding. It consumes comparison values and paths but does not compute metrics, validate command options, call providers, or define process output contracts.

# Connections

- [Image comparison](internal/imagediff/AGENTS.md): provides `ImageComparison` and image artifact paths.
- [Output](internal/output/AGENTS.md): provides safe filesystem writing.
- [Command orchestration](internal/pixelperfectcmd/AGENTS.md): assembles report input after command processing.

# Landmarks

- `internal/pixelperfectreport/report.go:Render`: renders report bytes from comparison input.
- `internal/pixelperfectreport/report.go:Write`: persists a report artifact.

# Boundary flows

- Information flow: `internal/imagediff/image.go:CompareImagesWithThresholds` -> `internal/pixelperfectreport/report.go:Render` via `internal/pixelperfectcmd/command.go:NewCommand`; value: `imagediff.ImageComparison`.

# Placement

Keep report templates and presentation transformations here. Add new evidence to its owning producer first, then render it without duplicating computation.