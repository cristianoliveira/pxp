# pxp command contracts

`pxp` emits structured results on stdout and diagnostics on stderr.

## Output

- TOON is the default structured format.
- `--json` selects compatibility JSON.
- Comparison, probe, and scan results include bounded output by default.
- Use `--full` or the relevant limit flag to request complete diagnostic output.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | The command completed and no configured validation gate failed. |
| `1` | A validation gate or operational check failed. |
| `2` | Command arguments or options were invalid. |

## Examples

```bash
pxp reference.png actual.png --json
pxp probe reference.png actual.png --at 20,20 --format json
pxp scan reference.png actual.png --row 20 --format json
pxp review reference.png actual.png --out .pxp-review
```

## Local annotated review loop

`pxp review` copies both PNGs into an immutable round directory, creates the
mask and directional overlay, and starts a short-lived server bound to
`127.0.0.1`. Open the URL printed on stderr. The page shows reference, actual,
and overlay images. Click or drag on the actual image to add point or rectangle
notes; display scaling is converted back to original image pixels.

Use **Submit feedback** to finish a round with `decision: submitted`, or use
**Approve** to finish with `decision: approved`. The structured result on stdout
contains `decision`, `feedback_path`, and the snapshot hashes. Feedback is
written once and never overwritten.

A complete multi-round loop is:

```bash
# Round 1: human annotates, then clicks Submit feedback.
pxp review reference.png actual-v1.png --out .pxp-review --json > round-1-result.json
# Agent reads feedback_path, fixes the implementation, and captures actual-v2.png.
pxp review reference.png actual-v2.png --out .pxp-review \
  --previous-feedback "$(jq -r .feedback_path round-1-result.json)" \
  --json > round-2-result.json
# Repeat until the human clicks Approve.
```

The POC is intentionally single-user and local: it has no authentication,
collaboration, freehand drawing, live editing, or automatic agent edits. Keep
review artifact directories private and treat feedback as local project data.

The result remains machine-readable when a validation gate fails. The process
exit code is the signal for automation.
