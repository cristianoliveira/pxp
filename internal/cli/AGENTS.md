# Purpose

`internal/cli` owns shared process-facing helpers: output-printer wiring, usage classification, safe error envelopes, recovery hints, executable-path display, and exit-code mapping.

# Boundaries

This package is runtime glue, not an image-analysis capability. It may depend on [structured output](internal/output/AGENTS.md), but it must not own comparison policy or provider behavior.

# Connections

- [Output](internal/output/AGENTS.md): supplies the `output.Printer` used for structured diagnostics and results.
- [Command orchestration](internal/pixelperfectcmd/AGENTS.md): supplies command errors and receives shared CLI behavior.
- [Executable composition](cmd/AGENTS.md): uses `RenderError` and `ExitCode` at process exit.

# Landmarks

- `internal/cli/runtime.go:NewPrinter`: binds the selected output format to a Cobra command.
- `internal/cli/error_output.go:RenderError`: emits the process-level error contract.
- `internal/cli/runtime.go:ExitCode`: maps typed failures to process status.

# Boundary flows

- Information flow: `internal/pixelperfectcmd/command.go:NewCommand` -> `internal/cli/error_output.go:RenderError` via `cmd/pxp/main.go:main`; value: `error`.

# Placement

Keep cross-command process policy here. Put domain-specific validation and result construction in the owning capability.