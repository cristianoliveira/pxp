# Purpose

`cmd/` owns the `pxp` executable composition root.

# Boundaries

Keep command wiring thin. Reusable image analysis belongs in `internal/imagediff`; workflow policy belongs in `internal/pixelperfectcmd`; report rendering belongs in `internal/pixelperfectreport`.

# Landmark

- `cmd/pxp/main.go:main`: process entrypoint for the standalone screenshot comparison CLI.
