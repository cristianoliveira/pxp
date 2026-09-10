# Purpose

`internal/cli` contains shared `pxp` process helpers: structured output, usage classification, error rendering, exit codes, and executable-path display.

# Boundaries

This package is composition glue, not an image-analysis capability. Keep comparison policy in `internal/pixelperfectcmd` and metrics in `internal/imagediff`.
