# Purpose

`pxp` is an agent-facing CLI for deterministic PNG comparison and visual regression evidence.

# Architecture

`cmd/pxp` owns executable composition. `internal/pixelperfectcmd` owns Cobra workflow policy. `internal/imagediff` owns deterministic image metrics and artifacts; `internal/imagecontext` owns optional advisory provider calls; `internal/annotations` owns annotation validation; `internal/pixelperfectreport` owns HTML reports; `internal/output` owns structured output and files.

# Boundary flow

`cmd/pxp/main.go` -> `internal/pixelperfectcmd.NewCommand` -> `internal/imagediff` and `internal/pixelperfectreport`.

Keep provider calls and filesystem wiring at the edges. Keep image metrics deterministic and independent of external services.
