# TASK-0025 anatomy workflow decision

Status: **provisional until independent Kelly evidence is attached**.

## Decision

**Retain the anatomy workflow as an optional geometry-evidence step.** Do not
promote its output to semantic UI recognition. The offline controls show that
it answers absolute bounds questions in one command and that bounded output can
be recovered with the exact hint. Existing comparison, probe, and scan remain
useful controls for different questions; they do not replace element discovery.

## Deterministic control results

The harness was run with the current `pxp` binary and six immutable cases. It
uses one bounded command per workflow and records fixture/prompt hashes in its
untracked report. Geometry checks are authored separately from prompts.

| case | anatomy result | geometry check | anatomy bytes | anatomy truncation |
| --- | --- | --- | ---: | --- |
| simple-control | exit 0 | pass | 587 | no |
| grouped-control | exit 0 | pass | 418 | no |
| empty-control | exit 0 | pass | 286 | no |
| truncated-control | exit 0 | pass | 527 | yes; `--full` hint |
| malformed-control | exit 1 | pass; recovery JSON | 175 | no |
| ambiguous-layout-holdout | exit 0 | provisional anchor pass | 4513 | yes; bounded at 20 |

The simple, grouped, empty, truncation, and malformed checks are exact control
assertions. The ambiguous holdout checks only geometry anchors and dimensions;
its expected anchors are not evidence of semantic roles.

For the simple control, one-command setup output sizes were: anatomy 587 bytes,
comparison 543, probe 515, scan 474, and review help 1076. These are setup-cost
controls, not a claim that the alternatives solve the same geometry problem.
Review was not launched because it requires an interactive browser decision.

## Agent-facing observations

### Kelly — independent blinded evaluator

Kelly used the holdout without reading `expected/ambiguous-layout.json` or prior
reports. This is qualitative evidence, not accepted-answer scoring. Three
bounded commands were used:

1. `pxp anatomy holdout/ambiguous-layout.png --json --limit 8` — 1,940 bytes;
   436x406, `#FFFFFF` background, `total=232`, `returned=8`, truncation, and an
   exact `--full` hint.
2. `pxp anatomy holdout/ambiguous-layout.png --json --group 8 --min-pixels 4 --limit 8`
   — 1,453 bytes; `total=22`, `returned=8`, truncation, and recovery hint.
3. `pxp anatomy holdout/ambiguous-layout.png --json --background nope` — exit 2
   with a named corrective error.

Kelly found one-round geometry evidence useful. Raw components were glyph-like;
`group` and `min-pixels` were tuning heuristics, not semantic labels. Returned
bounds were in-image and reading-order sorted, and density matched the emitted
pixel/area formula. No OCR, provider, network, or semantic overclaim was used.

### Implementer — holdout exploration

I used the same holdout before reading expected anchors. The first command was
`pxp anatomy ... --json --limit 12` (2,718 bytes, 232 total, 12 returned),
followed by `--group 8 --min-pixels 8 --limit 20` (4,465 bytes, 22 total,
20 returned). The first result was too glyph-like for layout decisions; the
second exposed repeated row geometry but still required a follow-up scan to
answer edge/color questions. This observation is implementer-biased and is not
independent acceptance evidence.

Both observations suggest the workflow should call regions “foreground
components” or “bounds”, never buttons, rows, icons, or text without separate
measurement or domain evidence.

## Follow-ups (not implemented here)

- Consider an explicit semantic-neutral field name or prose warning in bounded
  output if independent users repeatedly call regions “buttons” or “rows”.
- Consider a compact summary mode for large screenshots only after measuring
  whether it preserves actionable bounds.
- Keep accepted geometry, prompts, and generated reports in separate paths.
