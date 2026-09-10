# Purpose

`internal/imagediff` owns deterministic image evidence over already-decoded images: threshold and perceptual metrics, masks, mismatch regions, classifications, overlays, probes, scans, and movement/offset suggestions.

# Boundaries

The package is the source of truth for measurements. It does not open files, encode PNGs, parse Cobra flags, call visual providers, render HTML, or decide command exit policy. File/path adapters live in [imageio](internal/imageio/AGENTS.md).

# Connections

- [Annotations](internal/annotations/AGENTS.md): provides semantic intersections for region enrichment.
- [Image I/O](internal/imageio/AGENTS.md): decodes image files and persists masks/overlays at the edge.
- [Command orchestration](internal/commands/AGENTS.md): validates inputs and coordinates analysis.
- [Visual context](internal/imagecontext/AGENTS.md): consumes region evidence as an advisory input, never as a metrics authority.

# Landmarks

- `internal/imagediff/image.go:DecodedImages.Compare`: performs a complete deterministic in-memory comparison and returns mask pixels.
- `internal/imagediff/region_metrics.go:DecodedImages.MeasureRegions`: measures bounded regions.
- `internal/imagediff/offset.go:DecodedImages.SuggestOffset`: produces advisory translation evidence.
- `internal/imagediff/overlay.go:DecodedImages.Overlay`: produces an in-memory comparison overlay.
- `internal/imagediff/probe.go:DecodedImages.Probe`: measures normalized RGBA values at a point.
- `internal/imagediff/scan.go:DecodedImages.Scan`: produces deterministic horizontal or vertical color runs.

# Placement

Put new image-data evidence here when it remains deterministic, in-memory, and provider-independent. Put path/codec work in `imageio`; keep option semantics and adapters at their boundaries.
