# Release notes

## Next release

### Breaking CLI change

The comparison command no longer accepts `--report` or writes static HTML
reports. Comparison remains a non-interactive, structured metrics workflow and
continues to write masks, overlays, crops, and other requested image evidence.

For annotated human review, migrate to:

```bash
pxp review reference.png actual.png --out .pxp-review --json
```

The command waits for the explicit **Submit feedback** or **Approve** decision.
Legacy `--report` invocations fail with a migration diagnostic pointing to this
command; the flag is not silently ignored.
