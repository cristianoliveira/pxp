---
name: pxp
description: >
  Implement or refine a UI component from a reference screenshot by
  harnessing image comparison with the pxp CLI. Use for requests
  like "build this component from this PNG", "make this component match", or
  "pixel perfect this UI" when application code changes are expected.
  Also supports screenshot comparison without code changes when explicitly requested.
  Not for implementing a UI without a visual reference.
---

# Harness image comparison

Turn a reference screenshot into a working component in the user's application.
The deliverable is real code, verified behavior, and measured visual progress—not just a diff report.

## 1. Set up

- Locate the app, reference image, and target component. Read local project guidance and reuse existing components, styles, and assets.
- Check `pxp --help` and the available browser capture tool. Use the project's browser tooling or `playwright-cli`; check its help rather than guessing commands.
- Confirm reference dimensions, visible content, expected interactions, and any supplied acceptance gate. Ask only about missing information that blocks implementation. A PNG alone does not require external access.
- If reference or capture tooling is unavailable, report the blocker. Do not invent visual measurements or claim a verified match.
- Keep artifacts in the project's temporary/output directory, separate from production assets. Never render the reference image as the implementation.

## 2. Implement a first version

- Add or update the real component, with explicit props/state and the app's normal styling conventions. Use semantic HTML, accessible controls, and behavior tests that follow local patterns.
- Render that same component in an isolated temporary page, route, story, or preview. Import its real styles; do not maintain a second screenshot-only implementation.
- Supply fixed example data. Do not depend on live uploads, clocks, random IDs, or network responses for capture state.
- Capture at the reference's native scale. Fix viewport, device scale, browser, background, and shadow padding; wait for fonts and images, and disable animation and carets.
- Save this first render as the baseline before visual tuning.

## 3. Measure and refine

```bash
pxp reference.png actual.png \
  --threshold 8 --json \
  --output diff-mask.png --overlay diff-overlay.png \
  --report diff-report.html > metrics.json
```

Use separate filenames per iteration so the baseline survives.

1. Compare equivalent regions. Never resize images to force equal dimensions. Use image export metadata when supplied, or explicit CLI crops for known capture padding; do not crop away component defects.
2. Inspect the overlay and largest actionable regions. Fix outer geometry and spacing first, then typography, colors, icons, and effects.
3. Form one hypothesis from pixels and DOM facts. Make one bounded change, capture again under the same conditions, and repeat the same comparison.
4. Retain changes that improve the target without breaking behavior or other affected components. Revert or revise regressions.

Use `pxp scan` for edge/spacing questions and `pxp probe` for exact colors. Probe defaults to 25 points and scan defaults to 25 runs per image. Read [measurement details](references/measurement.md) only when you need crop syntax, metric interpretation, or diagnostic commands. Consult command help for other flags; do not explore every CLI feature before implementing.

Default budget: **five measured refinement iterations**, unless the user sets another limit. Stop earlier when the supplied gate passes, or after two consecutive iterations without improvement. Keep the best version; do not loop indefinitely.

## 4. Verify and deliver

- Check the component in its intended app context, not only the isolated preview. If only a reusable component was requested, leave a working preview and explain how to use it.
- Run relevant tests and build checks. For interactive components, check keyboard access, visible focus, accessible names, and requested actions. Do not invent backend integration from a screenshot; keep demo transitions explicit.
- Remove temporary scaffolding unless needed to reproduce the result. Preserve the capture instructions and evidence paths.
- Report changed files, baseline versus final metrics for the same region/settings, checks run, and remaining mismatches or blockers.
- A lower error is progress, not proof of a match. Claim completion against the user's gate only when it passes. Without a gate, describe measured improvement and remaining differences; never silently invent a tolerance or dismiss residual error as rasterization noise without evidence.

## 5. Human review (final)

- Produce a HTML report with a diff overlay and metrics.
- For an annotated, multi-round human handoff, use the `pxp-review-loop`
  skill and `pxp review`; keep Submit feedback distinct from Approve.
- A form for feedback, on submit download a report for you to continue.
