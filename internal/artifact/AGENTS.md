# Purpose

`internal/artifact` owns generic filesystem creation for generated `pxp` artifacts.

# Boundaries

It creates parent directories and writes bytes or open files with caller-selected permissions. It does not know artifact formats, image semantics, reports, or command policy.

# Connections

- [Image I/O](internal/imageio/AGENTS.md): uses file creation when encoding PNG artifacts.
- [Annotation persistence](internal/annotationio/AGENTS.md): uses file creation for annotation JSON.
- [Reports](internal/report/AGENTS.md): uses byte persistence for HTML reports.
- [Review](internal/review/AGENTS.md): supports immutable review snapshot artifacts through its image and feedback workflows.

# Landmarks

- `internal/artifact/file.go:CreateFile`: creates parent directories and opens an artifact for writing.
- `internal/artifact/file.go:WriteFile`: writes bytes while creating missing parent directories.

# Placement

Put format-neutral filesystem operations here. Keep encoding, serialization, and artifact naming in the owning adapter or presentation package.
