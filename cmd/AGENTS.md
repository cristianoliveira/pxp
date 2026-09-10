# Purpose

`cmd/` contains executable composition roots. The `pxp` command is the repository's standalone process entrypoint.

# Boundaries

This layer creates the command, configures process-level behavior, renders terminal errors, and exits with the command's status. It does not implement comparison policy or image algorithms.

# Connections

- [Command orchestration](internal/commands/AGENTS.md): provides the Cobra command tree consumed by `main`.
- [CLI runtime](internal/cli/AGENTS.md): provides error rendering and exit-code mapping used at process exit.

# Landmarks

- `cmd/pxp/main.go:main`: executes the command and maps returned errors to structured diagnostics and an exit code.

# Boundary flows

- Information flow: `internal/commands/command.go:NewCommand` -> `internal/cli/error_output.go:RenderError` via `cmd/pxp/main.go:main`; value: `error`.

# Placement

Keep executable wiring here. Put reusable runtime behavior in `internal/`; add another executable only when it has a distinct process contract.