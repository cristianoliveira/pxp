# pxp

`pxp` is an offline, agent-facing CLI for comparing PNG screenshots. It reports
raw and perceptual image metrics, mismatch regions, probes, scans, overlays, and
self-contained HTML reports.

## Install

Requires Go 1.25.5 or newer:

```bash
go build -o bin/pxp ./cmd/pxp
go install ./cmd/pxp
```

With Nix:

```bash
nix run .#pxp -- --help
nix build .#pxp
```

## Usage

```bash
pxp reference.png actual.png --overlay overlay.png --report visual-diff.html
pxp probe reference.png actual.png --at 20,20
pxp scan reference.png actual.png --row 20
```

TOON is the default structured output; use `--json` for compatibility JSON.
Comparison does not fail solely because pixels differ. Set `--max-*` validation
limits when the command should enforce a visual regression gate.

See [the command guide](cmd/pxp/README.md) for crops, masks, profiles,
annotations, visual context, metrics, and exit codes.

## Development

```bash
make test
make build
```

The project has one executable: `pxp`.
