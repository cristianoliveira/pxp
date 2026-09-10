# Purpose

`internal/imagediff` owns generic PNG comparison: decoding, thresholds, masks, overlays, region metrics, movement/offset evidence, and mismatch classification. It has no external dependency.

# Boundaries

Comparison results are deterministic evidence. The package may consume annotation matches for enrichment and write image artifacts, but it does not call providers, parse Cobra flags, or render HTML reports.

# Connections

- [Annotations](internal/annotations/AGENTS.md): supplies semantic region intersections for optional enrichment.
- [Output](internal/output/AGENTS.md): supplies deterministic artifact file creation.
- [pxp orchestration](internal/pixelperfectcmd/AGENTS.md): validates options and coordinates comparison operations.
- [Visual context](internal/imagecontext/AGENTS.md): is an optional consumer of region evidence, not a metrics authority.

# Landmarks

- `internal/imagediff/image.go:CompareImagesWithThresholds`: full deterministic comparison entrypoint.
- `internal/imagediff/region_metrics.go:MeasureImageRegionWithThresholds`: bounded regional metrics.
- `internal/imagediff/offset.go:SuggestImageOffset`: advisory translation evidence.
- `internal/imagediff/overlay.go:WriteImageOverlay`: comparison artifact output.

# Boundary flows

- Information flow: `internal/imagediff/image.go:CompareImagesWithThresholds` -> `internal/imagecontext/client.go:Client` via `internal/pixelperfectcmd/command.go:NewCommand`; value: `imagediff.ImageComparison.Regions`.

# Placement

Put new evidence or image algorithms here when they operate on image data without  or CLI policy. Keep provider calls and user-facing option semantics at their boundaries.
