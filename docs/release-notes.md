# Release notes

## v0.2.0 (unreleased)

`v0.2.0` makes `pxp review` the single human-review workflow. Comparison
remains non-interactive and machine-readable.

### New capabilities

- Compare screenshots with bounded TOON or JSON metrics.
- Inspect mismatch regions with masks, overlays, probes, and scans.
- Review reference, actual, and overlay images in a local browser.
- Save immutable review rounds with context, annotations, snapshot hashes, and
  feedback for the next round.
- Use numbered annotation markers and compact pin notes.
- Run coverage, complexity, and browser behavior quality gates.

### Breaking change

The comparison `--report` flag was removed. Comparison no longer writes static
HTML reports. Legacy invocations fail with a usage diagnostic.

```text
--report was removed; comparison no longer writes HTML reports
Use `pxp review <reference.png> <actual.png>` for human review, or omit --report for structured comparison.
```

For human review, run:

```bash
pxp review reference.png actual.png --out .pxp-review --json
```

Wait for **Submit feedback** or **Approve**. For automation, run comparison
without `--report` and consume its TOON output or `--json` output.

### Release preparation

The release workflow runs `go test ./...`, builds six platform archives, and
writes `checksums.txt`. Prepare and verify the release before publishing:

```bash
make test
make quality
git diff --check
```

After explicit release approval, create and push the tag:

```bash
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

Do not run the tag commands until the release is approved.
