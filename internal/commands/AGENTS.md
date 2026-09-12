# Purpose

`internal/commands` owns the `pxp` command workflow: Cobra command construction, option validation, crop and metadata preparation, deterministic comparison, probes, scans, annotations, optional visual context, validation gates, and artifact orchestration.

# Boundaries

It coordinates capabilities but does not implement image algorithms, provider protocols, HTML templates, or process exit handling.

# Connections

- [Image comparison](internal/imagediff/AGENTS.md): performs deterministic measurements and image artifacts.
- [Annotations](internal/annotations/AGENTS.md): loads and matches semantic region metadata.
- [Visual context](internal/imagecontext/AGENTS.md): provides optional provider-backed descriptions.
- [Reports](internal/report/AGENTS.md): renders HTML from comparison inputs and results.
- [Output](internal/output/AGENTS.md): emits structured command results.
- [CLI runtime](internal/cli/AGENTS.md): supplies shared printer and error policy.

# Landmarks

- `internal/commands/command.go:NewCommand`: creates the command tree and binds comparison, probe, and scan workflows.

# Boundary flows

- Information flow: `internal/imageio/imageio.go:CompareImagesWithThresholds` -> `internal/report/report.go:Render` via `internal/commands/command.go:NewCommand`; value: `imagediff.ImageComparison`.
- Information flow: `internal/imageio/imageio.go:CompareImagesWithThresholds` -> `internal/output/printer.go:Printer.Structured` via `internal/commands/command.go:NewCommand`; value: `imagediff.ImageComparison`.
- Information flow: `internal/imageio/imageio.go:CompareImagesWithThresholds` -> `internal/imagecontext/openrouter.go:OpenRouter.Describe` via `internal/commands/command.go:NewCommand`; value: `[]imagecontext.Region`.

# Placement

Put command option semantics and sequencing here. Put reusable evidence in `imagediff`, provider adapters in `imagecontext`, report presentation in `report`, and process behavior in `cli`.