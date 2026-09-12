# pxp command contracts

`pxp` emits structured results on stdout and diagnostics on stderr.

The [review-loop vision and goals](review-loop-vision.md) describe the intended human–agent review experience, not the current command contract.

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
```

The result remains machine-readable when a validation gate fails. The process
exit code is the signal for automation.
