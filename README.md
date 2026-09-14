# pxp - Pixel Perfect

You give a coding agent a screenshot and ask it to build the UI. The result looks close, but the spacing is off, a color doesn't match, and now you need to explain what to fix in plain english, good luck.

`pxp` compares the reference PNG with a screenshot of your implementation.
It shows where they differ, gives you numbers to compare between changes, and generates masks and overlays for inspection.

The idea is to give the agent something more useful than “it still looks wrong”.
Comparison runs locally, without an API key or a vision model.

## What you get

- Regions where the images differ, so you can focus on a smaller area.
- Raw and perceptual metrics to check what changed between iterations.
- Pixel probes and row or column scans for questions about colors and spacing.
- Masks and overlays with the images and metrics needed for machine-readable comparison.
- Optional metric limits when you want a comparison to fail in CI.

The output is meant for agents too. Comparison results use TOON by default, with JSON available through `--json`.
Results are bounded so the agent doesn't need to read every mismatch at once.
Optional visual descriptions can add context, but they don't change the measurements.

## Install

### Homebrew

On macOS, install the latest release from the tap:

```bash
brew tap cristianoliveira/tap
brew install pxp
```

### Nix

Run the packaged release directly from Cristian's Nix packages:

```bash
nix run github:cristianoliveira/nixpkgs#pxp -- --help
nix profile install github:cristianoliveira/nixpkgs#pxp
```

Or use the repository's development flake from a local checkout:

```bash
nix run .#pxp -- --help
nix build .#pxp
```

### GitHub Releases

Download an archive from [GitHub Releases](https://github.com/cristianoliveira/pxp/releases) for your OS (`linux`, `darwin` for macOS, or `windows`) and CPU (`amd64` or `arm64`). Extract it and put `pxp` (or `pxp.exe`) on your `PATH`. Each release includes `checksums.txt` with SHA-256 hashes for the archives.

From a local checkout, with Go 1.25.5 or newer:

```bash
go install ./cmd/pxp
```

Make sure your Go binary directory is on `PATH`. Or build a local executable:

```bash
go build -o bin/pxp ./cmd/pxp
./bin/pxp --help
./bin/pxp --version
```

`--version` reports the build version. Local `make build` uses `git describe`;
release archives use their pushed tag.

## Try it

Capture your UI and compare it with the reference:

```bash
pxp reference.png actual.png --overlay overlay.png --json > metrics.json
```

This prints structured metrics and writes `actual.diff.png` and `overlay.png`.
Keep the previous capture so you can check whether the change helped.

For annotated human review, run:

```bash
pxp review reference.png actual-v1.png --out .pxp-review --json > round-1.json
# After the human submits notes and the agent fixes the UI:
pxp review reference.png actual-v2.png --out .pxp-review \
  --previous-feedback "$(jq -r .feedback_path round-1.json)" --json > round-2.json
```

The command waits for **Submit feedback** or **Approve**. Submit feedback to
request another round. Approve to end the loop. Pins and rectangles use
original-image pixel coordinates. Each round keeps immutable snapshot hashes.

For a closer look:

```bash
# Check the colors at a point.
pxp probe reference.png actual.png --at 20,20

# Check color runs across a row.
pxp scan reference.png actual.png --row 20

# Save comparison metrics as JSON.
pxp reference.png actual.png --json > metrics.json
```

Use coordinates inside your images.
The [command guide](cmd/pxp/README.md) covers crops, masks, profiles, annotations, validation gates, and exit codes.

## A few constraints

PXP doesn't take screenshots, resize images, or align them automatically.
You need to supply images with equal dimensions after any crops (use explicit crops when the captures include different surrounding areas).
Keep the browser, scale, fonts, and capture state consistent between comparisons.

The thing is, a smaller error doesn't tell you whether a button works, or whether the UI actually looks right.
You still need to check those things. The numbers help you investigate; they aren't an approval.

Finding different pixels doesn't fail the command by itself.
Set `--max-*` limits if you need a pass/fail check, using tolerances that make sense for your captures.

## Breaking CLI change

The comparison `--report` flag was removed. Comparison remains non-interactive
and emits structured metrics; use `pxp review reference.png actual.png` for the
blocking annotated human-review workflow. Legacy `--report` invocations fail
with a migration diagnostic instead of being silently ignored.

## Using it with an agent

The [PXP skill](skills/pxp/SKILL.md) describes the workflow: build the real component, capture a baseline, compare, and make a limited number of changes.
It also asks the agent to report what still differs, rather than call it done just because the score improved.

## Releasing

The current published release is `v0.1.0`. The release workflow runs when a
`v*` tag is pushed. It tests the project, builds Linux, macOS, and Windows
archives for `amd64` and `arm64`, and publishes SHA-256 checksums as GitHub
Release assets. It keeps the build artifacts for seven days.

Prepare `v0.2.0` with the checks below. Do not create the tag until the release
is approved.

```bash
make test
make quality
git diff --check
```

After approval, publish the release:

```bash
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

A tag such as `v0.2.0-rc.1` creates a prerelease.

## Development

### Everyday checks

Run the fast checks while changing the code:

```bash
make test
make build
make fmt
make vet
```

### Advanced checks

Run these checks before release or compatibility review:

```bash
make quality
make quality-test
make test-race
```

`make quality` requires 78.0% statement coverage. It limits complexity to 40
and function size to 203 lines or 127 statements. The browser watcher uses
`playwright-cli` 0.1.9 and Chromium. See [the quality guardrails](docs/quality-guardrails.md)
for details.
