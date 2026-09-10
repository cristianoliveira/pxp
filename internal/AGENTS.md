# Purpose

`internal/` contains the private capabilities behind `pxp`.

# Boundaries

- `internal/pixelperfectcmd` owns command options and orchestration.
- `internal/imagediff` owns deterministic PNG comparison.
- `internal/imagecontext` owns optional visual descriptions.
- `internal/annotations` owns annotation data and intersection math.
- `internal/pixelperfectreport` owns HTML report rendering.
- `internal/output` owns structured output and filesystem artifacts.
- `internal/cli` owns shared Cobra error and output helpers.

Keep external integrations at the edge and image metrics independent of them.
