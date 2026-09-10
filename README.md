# pxp - Pixel Perfect

**Stop guessing what changed. Measure it.**

A UI can look close to a reference screenshot and still have the wrong spacing,
colors, or alignment. Pixel Perfect (`pxp`) compares two PNGs and gives you—and
your coding agent—evidence for the next fix.

Get mismatch regions, image metrics, a diff overlay, and a self-contained HTML
report from one command. Core comparison runs offline. No API key or vision
model required.

## From “looks close” to a measured next step

- **Find where to look.** Mismatch regions and overlays show where images differ.
- **Inspect the details.** Probe exact pixel colors or scan rows and columns to
  investigate spacing and edges.
- **Measure progress.** Raw and perceptual metrics let you compare iterations
  under the same settings.
- **Share the evidence.** HTML reports put images, metrics, and provenance in one
  file for review.
- **Set your own gate.** Use explicit metric limits to enforce visual regression
  checks in CI.

Built for agent workflows: deterministic measurements, bounded results, TOON-first
structured output, and JSON when you need it. Optional visual descriptions add
context; they do not change the measurements.

## Install

From a local checkout, with Go 1.25.5 or newer:

```bash
go install ./cmd/pxp
```

Make sure your Go binary directory is on `PATH`. Or build a local executable:

```bash
go build -o bin/pxp ./cmd/pxp
./bin/pxp --help
```

With Nix:

```bash
nix run .#pxp -- --help
nix build .#pxp
```

## Compare your first screenshots

Capture your UI, then compare it with the reference:

```bash
pxp reference.png actual.png --overlay overlay.png --report visual-diff.html
```

You get structured metrics in the terminal, a changed-pixel mask at
`actual.diff.png`, an overlay, and an HTML report. Open `visual-diff.html` to
review the evidence, fix one issue, capture again, and compare with the same
settings.

Need a closer look?

```bash
# Inspect exact colors at a point.
pxp probe reference.png actual.png --at 20,20

# Inspect color runs across a row.
pxp scan reference.png actual.png --row 20

# Save comparison metrics as JSON.
pxp reference.png actual.png --json > metrics.json
```

Use coordinates inside your images. See the [command guide](cmd/pxp/README.md)
for crops, masks, profiles, annotations, validation gates, and exit codes.

## Evidence, not a promise of perfection

PXP compares screenshots; it does not capture them, resize them, or align them
automatically. Prepared inputs must have equal dimensions. Use explicit crops
when captures include different surrounding areas.

A lower error is progress—not proof that a UI looks right or works correctly.
Keep capture conditions fixed and review behavior separately. Differences alone
do not fail the command; configure `--max-*` limits when you need a pass/fail gate.

## Use it with a coding agent

The [PXP skill](skills/pxp/SKILL.md) guides screenshot-driven UI implementation:
build a real component, capture a baseline, make bounded refinements, and report
measured progress and remaining differences.

## Development

```bash
make test
make build
```
