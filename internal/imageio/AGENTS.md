# Purpose

`internal/imageio` owns PNG decoding, dimension checks, and filesystem adapters around [deterministic image evidence](internal/imagediff/AGENTS.md).

# Boundaries

It converts image files to in-memory comparison inputs and persists PNG outputs. It does not define comparison algorithms, command flags, or provider behavior.

# Connections

- [Image comparison](internal/imagediff/AGENTS.md): provides decoded images and consumes its comparison, measurement, and overlay operations.
- [Artifact persistence](internal/artifact/AGENTS.md): provides filesystem creation for PNG outputs.
- [Command orchestration](internal/commands/AGENTS.md): invokes file-based comparison and inspection workflows.
- [Review](internal/review/AGENTS.md): supplies decoded images and persists review snapshot images.

# Landmarks

- `internal/imageio/imageio.go:LoadDecodedImages`: loads and normalizes a reference/actual PNG pair.
- `internal/imageio/imageio.go:CompareImagesWithThresholds`: adapts file paths to deterministic thresholded comparison.
- `internal/imageio/imageio.go:WritePNG`: encodes an image as a PNG artifact.

# Boundary flows

- Information flow: `internal/imageio/imageio.go:LoadDecodedImages` -> `internal/imagediff/image.go:DecodedImages.Compare` via `internal/imageio/imageio.go:CompareImagesWithThresholds`; value: `*imagediff.DecodedImages`.

# Placement

Put PNG and path handling here. Put reusable in-memory evidence in `imagediff`, and generic file creation in `artifact`.
