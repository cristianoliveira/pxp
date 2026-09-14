# Purpose

`internal/commands` owns the `pxp` command workflow: Cobra command construction, option validation, crop and metadata preparation, deterministic comparison, probes, scans, annotations, optional visual context, validation gates, and artifact orchestration.

# Boundaries

It coordinates capabilities but does not implement image algorithms, provider protocols, HTML templates, or process exit handling.

# Connections

- [Image comparison](../imagediff/AGENTS.md): performs deterministic measurements and image artifacts.
- [Annotations](../annotations/AGENTS.md): loads and matches semantic region metadata.
- [Visual context](../imagecontext/AGENTS.md): provides optional provider-backed descriptions.
- [Output](../output/AGENTS.md): emits structured command results.
- [CLI runtime](../cli/AGENTS.md): supplies shared printer and error policy.

# Landmarks

- `internal/commands/command.go:NewCommand`: creates the command tree and binds comparison, probe, and scan workflows.

# Boundary flows

- Information flow: `internal/imageio/imageio.go:CompareImagesWithThresholds` -> `internal/output/printer.go:Printer.Structured` via `internal/commands/command.go:NewCommand`; value: `imagediff.ImageComparison`.
- Information flow: `internal/imageio/imageio.go:CompareImagesWithThresholds` -> `internal/imagecontext/openrouter.go:OpenRouter.Describe` via `internal/commands/command.go:NewCommand`; value: `[]imagecontext.Region`.

# Placement

Put command option semantics and sequencing here. Put reusable evidence in `imagediff`, provider adapters in `imagecontext`, and process behavior in `cli`.
