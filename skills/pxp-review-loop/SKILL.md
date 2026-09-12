---
name: pxp-review-loop
description: >
  Run a human-in-the-loop visual review loop with the pxp CLI. Use when a
  user wants to inspect a reference/actual screenshot comparison, annotate
  differences in a browser, submit location-aware feedback, repeat after
  implementation changes, or explicitly approve the visual result. Also use
  for launching a durable local review session and handing its structured
  feedback to an agent. Do not use for autonomous UI editing, generic
  screenshot comparison without human annotations, or claiming fixture-only
  changes were applied.
---

# PXP annotated review loop

Use this skill to coordinate one or more local review rounds:

```text
compare -> human annotates -> Submit feedback -> agent fixes -> recapture
-> next comparison -> repeat until Approve
```

The loop is a handoff protocol, not an autonomous editing engine. Keep the
human decision explicit and keep each comparison round immutable.

## Before starting

1. Locate the reference and actual PNGs. Confirm they have equal dimensions.
2. Locate the editable application source and the capture command, if any.
3. If the pair is only a fixture or artifact, say so. Do not claim that notes
   changed an implementation when no editable source is associated.
4. Use a new private artifact root for every independent review session. Keep
   review artifacts outside production assets.
5. Explain that the local server is unauthenticated and binds to loopback.

## Start a round

Run the command in the foreground with a unique output directory. Prefer
`--open` when a default browser is available; stdout is the completion result
and stderr announces the URL:

```bash
mkdir -p .tmp/pxp-review
pxp review reference.png actual.png \
  --out .tmp/pxp-review \
  --context-file review-context.json \
  --open --json > .tmp/pxp-review/round-1.result.json \
  2> .tmp/pxp-review/round-1.server.log
```

The command starts the loopback server, prints the URL, attempts to open the
default browser, and waits until the browser records Submit or Approve. Browser
opening is best-effort; if it fails, use the printed URL manually while the
foreground command remains pending. Never detach or background the server, and
do not ask the human to return to chat to announce completion. Once the command
returns, immediately consume stdout and read the structured result. If the
environment cannot keep an expected human wait alive, report that limitation
instead of inventing a submission or polling a detached process.

The page displays the optional implementation context before the evidence,
then reference, actual, and overlay images. Context is supplied with
`--context-file` as a JSON object containing optional `title`, `what_changed`,
`what_to_test`, `expected_outcome`, `limitations`, and `source_reference`
strings. It is escaped, length-limited, preserved in round provenance, and
never replaces human judgment. The human can add:

- general notes;
- point annotations; and
- rectangle annotations.

Each annotation records a stable ID, image identity, and original-pixel
coordinates. Display scaling and canvas borders are accounted for. **Submit
feedback** or **Approve** first opens an explicit confirmation dialog; no
feedback is persisted until the final action. Review the round, note/annotation
count, and consequence, then choose **Send feedback to agent** or **Approve and
finish**. Back or Escape preserves the draft and restores focus to the invoking
action. Submit requires at least one nonblank note or annotation. Approve must
contain neither notes nor annotations.

## Interpret completion

Read the structured result only after the process exits:

```bash
cat .tmp/pxp-review/round-1.result.json
jq .feedback_path .tmp/pxp-review/round-1.result.json
```

Expected decisions:

- `submitted`: read the feedback JSON, preserve the snapshot identity and
  annotations, then fix the associated source and capture a new actual image.
- `approved`: stop the loop and report the approval and feedback path.

A cancelled or closed round returns an operational error and does not write a
feedback JSON file. Treat missing feedback as an incomplete round, not as
approval. Invalid submissions return an error while the server remains alive;
the human can correct the form and retry.

## Continue a round

Use a new actual capture and link the previous feedback. Never overwrite a
prior round:

```bash
previous=$(jq -r .feedback_path .tmp/pxp-review/round-1.result.json)
pxp review reference.png actual-v2.png \
  --out .tmp/pxp-review \
  --previous-feedback "$previous" \
  --json > .tmp/pxp-review/round-2.result.json \
  2> .tmp/pxp-review/round-2.server.log
```

Repeat until the human chooses Approve. Check that each result has a distinct
feedback path and that linked previous feedback remains byte-identical.

## Report and boundaries

At the end of each round, report:

- decision and feedback path;
- round/session ID;
- snapshot input identities or hashes;
- note IDs and coordinates that require action;
- whether editable source was available; and
- the next command or reason the loop stopped.

Do not call a lower pixel difference approval. Do not edit source code during
an active review round unless the workflow explicitly hands the feedback back
to an implementation step. Keep accounts, collaboration, freehand drawing,
live editing, and autonomous agent edits out of this POC.

For implementation and measurement workflows without this handoff loop, use
the `pxp` skill instead.
