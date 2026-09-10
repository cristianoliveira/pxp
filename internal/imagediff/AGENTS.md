# Purpose

`internal/imagediff` owns deterministic PNG evidence: decoding, dimension checks, threshold and perceptual metrics, masks, mismatch regions, classifications, overlays, and movement/offset suggestions.

# Boundaries

The package is the source of truth for measurements. It does not parse Cobra flags, call visual providers, render HTML, or decide command exit policy. It may use annotation geometry and shared file creation for enrichment and artifacts.

# Connections

- [Annotations](internal/annotations/AGENTS.md): provides semantic intersections for region enrichment.
- [Output](internal/output/AGENTS.md): provides file creation used by image artifacts.
- [Command orchestration](internal/pixelperfectcmd/AGENTS.md): validates inputs and coordinates analysis.
- [Visual context](internal/imagecontext/AGENTS.md): consumes region evidence as an advisory input, never as a metrics authority.

# Landmarks

- `internal/imagediff/image.go:CompareImagesWithThresholds`: performs a complete deterministic comparison.
- `internal/imagediff/region_metrics.go:MeasureImageRegionWithThresholds`: measures a bounded region.
- `internal/imagediff/offset.go:SuggestImageOffset`: produces advisory translation evidence.
- `internal/imagediff/overlay.go:WriteImageOverlay`: writes a comparison overlay.

# Boundary flows

- Information flow: `internal/imagediff/image.go:CompareImagesWithThresholds` -> `internal/imagecontext/openrouter.go:OpenRouter.Describe` via `internal/pixelperfectcmd/command.go:NewCommand`; value: `[]imagecontext.Region`.

# Placement

Put new image-data evidence here when it remains deterministic and provider-independent. Keep option semantics and adapters at their boundaries.