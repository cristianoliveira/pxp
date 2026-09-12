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
pxp review reference.png actual.png --out .pxp-review
pxp review reference.png actual.png --out .pxp-review --open --json
pxp review reference.png actual.png --context-file review-context.json --json
```

## Local annotated review loop

The [review-loop vision and goals](review-loop-vision.md) define the intended human–agent experience and product acceptance boundaries.

`pxp review` copies both PNGs into an immutable round directory, creates the
mask and directional overlay, and starts a short-lived server bound to
`127.0.0.1`. The command prints the URL on stderr and waits in the foreground.
With `--open`, it attempts a platform default-browser opener after the server is
ready; opener failure is nonfatal and the printed URL remains the manual
fallback. The page shows reference, actual, and overlay images. Click or drag
on the actual image to add point or rectangle notes; display scaling is
converted back to original image pixels. A contextual annotation editor opens
for the selected geometry; Save commits the note and Cancel/Escape leaves the
annotation and draft unchanged. The editor shows source view and original-pixel
geometry, and its pending note survives browser reload until explicitly saved
or cancelled.


Use `--context-file` to provide an optional JSON object with these string
fields: `title`, `what_changed`, `what_to_test`, `expected_outcome`,
`limitations`, and `source_reference`. The top level must be one non-null JSON
object; fields must be strings. Unknown fields, null fields, trailing JSON,
files larger than 16 KiB, or fields larger than 4,000 UTF-8 bytes are rejected.
Field edges are trimmed while internal line breaks are preserved. An omitted
flag, an empty/whitespace-only context file, or blank context fields is
represented as `No implementation context was provided for this round.` The
exact normalized context is included in the session,
`snapshot.json`, result, and feedback; `previous_feedback` links context across
rounds without changing screenshot identity or human notes. Context is rendered
as escaped text in a keyboard-accessible, collapsible panel before the evidence.

Use **Submit feedback** or **Approve** to choose a round decision. A valid
choice first opens an explicit confirmation dialog; it does not persist feedback
or unblock the agent. Review the round, note/annotation count, and consequence,
then choose **Send feedback to agent** (`decision: submitted`) or **Approve and
finish** (`decision: approved`). Back or Escape preserves the draft and returns
focus to the invoking action. Submitted feedback must include at least one note
or annotation; approval must include neither. The structured result on stdout
contains `decision`, `feedback_path`, and the snapshot hashes.
Feedback is written once and never overwritten. Cancelling the command or
closing the server before a decision returns an operational error and does not
write feedback.

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
exit code is the signal for automation. Consume stdout after the foreground
command exits; do not background the server or ask the human to announce
completion in chat.
