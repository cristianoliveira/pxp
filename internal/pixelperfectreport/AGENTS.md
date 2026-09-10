# Purpose

`internal/pixelperfectreport` renders and writes self-contained HTML reports for deterministic image-comparison evidence.

# Boundaries

The report consumes a comparison result and artifact paths. It owns presentation and safe embedding, not comparison algorithms, CLI validation, or provider calls.

# Connections

- [Image comparison](internal/imagediff/AGENTS.md): provides `ImageComparison` and artifact paths.
- [Output](internal/output/AGENTS.md): provides filesystem writing for the report.
- [pxp orchestration](internal/pixelperfectcmd/AGENTS.md): supplies report input after command processing.

# Landmarks

- `internal/pixelperfectreport/report.go:Render`: renders the HTML report bytes.
- `internal/pixelperfectreport/report.go:Write`: persists a report artifact.

# Boundary flows

- Information flow: `internal/imagediff/image.go:CompareImagesWithThresholds` -> `internal/pixelperfectreport/report.go:Render` via `internal/pixelperfectcmd/command.go:NewCommand`; value: `imagediff.ImageComparison`.

# Placement

Keep report templates and report-specific presentation here. Add new evidence fields to the owning result package first, then render them here without duplicating computation.
