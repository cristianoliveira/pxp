# pxp

Harness image comparison between two PNG files and report what changed. Use the metrics, mismatch regions,
and image artifacts to debug a UI or check a visual regression in CI.

`pxp` works offline without external credentials or a vision model. It does
not capture screenshots, resize inputs, or align them automatically. Prepared
images must have equal dimensions. Optional movement suggestions and visual
descriptions do not change the measurements.

## Install

From the repository root, with Go 1.25.5 or newer:

```bash
go build -o bin/pxp ./cmd/pxp
export PATH="$PWD/bin:$PATH"
```

Or use Nix from the repository root:

```bash
nix run .#pxp -- --help
nix build .#pxp
```

See the [project README](../../README.md#install) for clone and setup instructions.

## Quick start

Supply a reference screenshot and a screenshot of your implementation:

```bash
pxp reference.png actual.png \
  --overlay overlay.png \
  --report visual-diff.html
```

This prints TOON metrics and writes:

- `actual.diff.png`: a transparent changed-pixel mask; override with `--output`.
- `overlay.png`: a directional overlay, with reference differences in red and
  actual differences in green.
- `visual-diff.html`: a self-contained report.

Use `--json` for compatibility JSON. Keep input and output paths separate.

**A difference alone does not fail the command.** Add `--max-*` flags to set a
pass/fail rule. Invalid inputs and file errors still fail without those flags.

Run `pxp --help` for comparison options, `pxp probe --help`
for pixel inspection, and `pxp scan --help` for row and column scans.

## Find the cause of a difference

Start with the summary, then request smaller areas:

```bash
# Compare one area: x,y,width,height.
pxp reference.png actual.png --region 10,10,100,40

# Read colors and channel deltas at exact points.
pxp probe reference.png actual.png --at 20,20 --at 30,20

# Read color runs across a row or down a column.
pxp scan reference.png actual.png --row 20
pxp scan reference.png actual.png --column 30
```

Choose coordinates inside your images. A region keeps coordinates in the prepared
image space; it does not move that region's origin to zero.

### Probes and scans

| Command | Default output | Default bound |
| --- | --- | --- |
| Comparison | TOON | 20 mismatch regions |
| `probe` | CSV | 25 points |
| `scan` | CSV | 25 color runs per image |

Comparison uses `--max-regions` or `--full`. Probe and scan use `--limit` or
`--full`; do not combine those two flags. Truncated probe/scan output gives exact
counts and a command to retrieve all results. Use `--format json` for their JSON
views, including RGBA values and coordinates in the original inputs.

Sample a line and its surrounding pixels with:

```bash
pxp probe reference.png actual.png \
  --from 10,20 --to 50,20 --step 4 --radius 1 --format json
```

Line endpoints are inclusive. `--step` samples every Nth point; `--radius` expands
selections into deduplicated squares. Selected points must be in bounds.
`--row` aliases `--y`; `--column` aliases `--x`.

## Metrics

| Field | Meaning |
| --- | --- |
| `changedPixels`, `changedRatio` | Pixels with a channel difference above `--threshold`, and their share of compared pixels. |
| `comparedPixels` | Number of pixels included after region and exclusion controls. |
| `rmse` | Normalized RGB or RGBA root mean square error, based on input transparency. |
| `rgbRmse`, `luminanceRmse`, `alphaRmse` | Color, brightness, and transparency errors. |
| `edgeRmse` | Difference in local visible-luminance gradients; useful for geometry checks. |
| `perceptualRmse` | OKLab HyAB color distance after alpha compositing. |
| `perceptualChangedPixels`, `perceptualChangedRatio` | Pixels above `--perceptual-threshold`, and their share of compared pixels. |
| `antialiasedPixels` | Changed pixels with neighborhood ramp evidence; reported, not removed. |
| `evidence` | Counts in raw-only, perceptual-only, and overlapping changed-pixel sets. |
| `bounds`, `changedRows` | The smallest changed rectangle and sorted changed row indexes. |
| `regions` | Local bounds, measurements, dominant color pairs, and classification hints. |

Region classifications are deterministic hints, not diagnoses of exact CSS:

- `solid-fill`: concentrated color change with mostly aligned edges.
- `geometry`: edge differences dominate.
- `sparse-raster`: sparse changes, often text or antialiasing.
- `mixed`: no dominant signal.

Inspect the underlying metrics and images before choosing a fix. A lower global
score does not prove that a component looks better or behaves correctly.

## Set validation gates

```bash
pxp reference.png actual.png --json \
  --threshold 8 \
  --perceptual-threshold 0.1 \
  --max-rmse 0.02 \
  --max-changed-ratio 0.01 \
  --max-perceptual-changed-ratio 0.005
```

These values are **examples, not recommended tolerances**.

- `--threshold` ignores raw channel differences at or below its value (0–255;
  default 0) when counting changed pixels.
- `--perceptual-threshold` controls perceptually changed pixels (default 0.1;
  finite, non-negative HyAB distance). It is not a guaranteed visibility boundary.
- `--max-*` flags set acceptance limits. They do not change measurements.

A configured gate fails only when the metric exceeds its maximum. Output contains
`validation.passed` and all failed metrics, even on failure. The process exits 1
and writes a diagnosis to stderr; stdout remains a structured comparison result.

### Calibrate before using CI

1. Capture the accepted implementation several times under fixed conditions.
2. Measure repeat-run noise for the metrics you will gate.
3. Capture a small change that your team considers a real regression.
4. Choose limits above observed noise and below that regression. If those ranges
   overlap, improve the capture setup or use more focused checks.
5. Commit the capture settings and comparison command with the visual test.

Recheck limits after browser, OS, or font changes. Do not select a tolerance after
seeing a failing result just to make it pass.

## Crop inputs explicitly

Use crops when screenshots include different surrounding areas:

```bash
pxp reference.png actual.png \
  --reference-crop 10,10,100,80 \
  --actual-crop 30,20,100,80
```

Crops use `x,y,width,height` in each original image. The cropped images must have
equal dimensions. Region, probe, and scan coordinates then refer to cropped space;
output records the original input coordinates. Pass the same crop options to
follow-up probes and scans.

`--reference-metadata` can apply a logical crop recorded by an image export.
Do not combine it with `--reference-crop`.

## Exclude known differences

```bash
pxp reference.png actual.png \
  --ignore-region 0,0,20,20 \
  --ignore-region 50,20,10,10 \
  --mask comparison-mask.png
```

`--ignore-region` is repeatable. The mask must match the prepared comparison image:
visible non-black pixels are included; black or transparent pixels are ignored.
Excluded pixels do not contribute to comparison metrics. Document exclusions so
real regressions are not hidden.

## Group regions and inspect movement

```bash
pxp reference.png actual.png \
  --suggest-offset 5 \
  --suggest-movement 12 \
  --region-gap 8 \
  --min-region-pixels 12
```

- `--suggest-offset` searches for a whole-image translation within the given radius.
- `--suggest-movement` reports up to five local translation candidates. `dx` and
  `dy` describe reference-to-actual movement; confidence reflects RMSE improvement.
- `--region-gap` groups nearby mismatch regions, such as letters in a text block.
- `--min-region-pixels` omits small regions from reporting.

These options do not realign inputs or change global metrics. Small-region
filtering changes reporting, not validation.

## Reuse settings with a profile

Save this as `pxp.json`:

```json
{
  "version": 1,
  "suggestOffset": 8,
  "regionGap": 2,
  "minRegionPixels": 4
}
```

```bash
pxp reference.png actual.png --profile pxp.json
pxp reference.png actual.png --profile pxp.json --suggest-offset 0
```

Precedence is **built-in defaults < profile < explicit flags**, even when a flag
sets a built-in default. Output records resolved values and their sources.

Version 1 supports only the three settings shown above. Keep thresholds, input
paths, crops, masks, and artifact paths on the command line. Unknown fields,
unsupported versions, and invalid values fail.

## Use coordinates and labels

Annotations are optional. A generic annotation file looks like:

```json
{
  "version": 1,
  "coordinateSpace": {"width": 1280, "height": 720},
  "annotations": [
    {
      "id": "sidebar-row",
      "label": "Selected sidebar row",
      "bounds": {"x": 20, "y": 76, "width": 248, "height": 56},
      "metadata": {"source": "capture"}
    }
  ]
}
```

Coordinate-space dimensions must match the prepared image. Intersecting labels
and overlap ratios appear on mismatch regions. Metadata is passed through, not
interpreted. Annotations do not change metrics or acceptance gates; invalid
annotation files still fail validation.

## Optional visual descriptions

Only enable this when sending the screenshots to a provider is acceptable:

```bash
pxp reference.png actual.png --visual-context
pxp reference.png actual.png \
  --visual-context --visual-context-provider openai
```

Descriptions are advisory. They do not change deterministic metrics,
classifications, or validation gates. Provider calls can cost money and send
private image content outside your machine.

| Provider | API key | Optional model and endpoint overrides |
| --- | --- | --- |
| OpenRouter (default) | `OPENROUTER_API_KEY` | `OPENROUTER_MEDIA_MODEL`, `OPENROUTER_BASE_URL` |
| OpenAI | `OPENAI_API_KEY` | `OPENAI_VISION_MODEL`, `OPENAI_BASE_URL` |

Use `--visual-context-model` for a per-call model override and
`--visual-context-prompt` to add a focus. Configuration can also come from
`~/.pi/agent/pi-spectacles.json`, or the file named by `PI_SPECTACLES_CONFIG`.
OpenRouter settings are top-level; OpenAI settings can use an `openai` object.
Keep credentials outside the repository.

Missing credentials add a disclaimer without failing an otherwise successful
comparison. Other configuration or provider errors can fail the command. Leave
visual context off when CI must depend only on local image evidence.

## Capture stable screenshots

Keep these fixed between captures:

- Browser engine, OS, viewport, zoom, and device scale (commonly DPR 1).
- Font files; wait for `document.fonts.ready`.
- Application data, interaction state, background, and transparency.
- Animations, transitions, carets, and cursor visibility.
- Capture bounds, including padding for shadows and other effects.

Equal dimensions do not prove equal coordinate systems. Check design/DOM bounds
and use offset suggestions to investigate alignment, not to hide it.

## Exit codes and limits

| Code | Meaning |
| --- | --- |
| `0` | Comparison completed and no configured gate failed. |
| `1` | A gate failed or an operational error occurred. |
| `2` | Invalid command arguments or options. |

Usage and operational errors are structured on stdout. Failed gates keep the
comparison result on stdout and their diagnosis on stderr. See
[command contracts](../../docs/command-contracts.md) for the shared output rules.

This tool measures raster differences. It cannot identify the exact CSS fix or
prove accessibility, interaction behavior, or human visual acceptance. Keep
browser behavior tests and visual review alongside image measurements.
