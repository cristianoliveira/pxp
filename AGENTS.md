# Purpose

`pxp` is an offline, agent-facing CLI that compares PNG screenshots and emits deterministic visual-regression evidence: metrics, mismatch regions, probes, scans, masks, overlays, and HTML reports.

# Architecture

- [Executable composition](cmd/AGENTS.md) owns process startup.
- [Command orchestration](internal/commands/AGENTS.md) owns Cobra policy and workflow sequencing.
- [Image comparison](internal/imagediff/AGENTS.md) owns deterministic image evidence.
- [Visual context](internal/imagecontext/AGENTS.md) owns optional provider-backed descriptions.
- [Annotations](internal/annotations/AGENTS.md) owns annotation contracts and geometry.
- [Reports](internal/report/AGENTS.md) owns HTML presentation.
- [Output](internal/output/AGENTS.md) owns structured rendering and artifact files.
- [CLI runtime](internal/cli/AGENTS.md) owns shared process and error behavior.
- [Skill workflows](skills/AGENTS.md) and [evaluation controls](tests/evals/AGENTS.md) are agent-facing, not runtime code.

Composition is wired at the executable and command boundaries. Provider calls stay optional and at the edge; deterministic image metrics do not depend on them.

# Modules

- [Commands](cmd/AGENTS.md): executable entrypoint and composition.
- [Internal capabilities](internal/AGENTS.md): private runtime packages.
- [Documentation](docs/AGENTS.md): user-facing command contracts.
- [Development scripts](scripts/AGENTS.md): support tooling.
- [Skills](skills/AGENTS.md): agent workflows and packaging.
- [Evaluation controls](tests/evals/AGENTS.md): offline workflow evaluation inputs.

# Landmarks

- `cmd/pxp/main.go:main`: process entrypoint; executes the composed Cobra command.
- `internal/commands/command.go:NewCommand`: creates the CLI command tree.
- `internal/imagediff/image.go:CompareImagesWithThresholds`: starts deterministic comparison.
- `internal/output/printer.go:Printer.Structured`: emits the default structured result.
- `internal/report/report.go:Render`: creates self-contained report bytes.

# Boundary flows

- Information flow: `internal/commands/command.go:NewCommand` -> `internal/cli/error_output.go:RenderError` via `cmd/pxp/main.go:main`; value: `error`.
- Information flow: `internal/annotations/annotations.go:Load` -> `internal/imagediff/image.go:CompareImagesWithThresholds` via `internal/commands/command.go:NewCommand`; value: `annotations.Document`.
- Information flow: `internal/imagediff/image.go:CompareImagesWithThresholds` -> `internal/report/report.go:Render` via `internal/commands/command.go:NewCommand`; value: `imagediff.ImageComparison`.
- Information flow: `internal/imagediff/image.go:CompareImagesWithThresholds` -> `internal/output/printer.go:Printer.Structured` via `internal/commands/command.go:NewCommand`; value: `imagediff.ImageComparison`.

# Placement

Put reusable image evidence in `imagediff`, command policy in `commands`, provider adapters in `imagecontext`, presentation in `report`, and shared output/process behavior in their respective guides. Add a new top-level module only when a cohesive responsibility has an independent owner and boundary.