# Upload-panel evaluation controls

This folder contains accepted evidence, deliberate defects, and independent checks
for the upload-panel implementation task. It is for graders, **not candidate
agents**.

Keep these files outside `skills/`. Never pass this folder through `--skill-path`
or `--file`. The skill runner copies skill content into candidate runs, so putting
an answer there would reveal the solution.

Folder separation is not a filesystem sandbox. For a true holdout, run candidates
in an environment that cannot read this directory. A run that can access the host
filesystem is not isolated from answers merely because they were not copied.

## Choose a check

| Goal | Tool | Requires |
| --- | --- | --- |
| Check fixture integrity and runner policy | Python unit tests | Python 3.10+; no browser or model calls. |
| Exercise accepted and defective implementations | `verify.py` | Node.js 22.12+, npm 11, `pxp`, and `playwright-cli` with Chromium. |
| Verify one candidate's delivered app | `ab_verify.py` | The browser/build toolchain and a completed run with `outputs/app`. |
| Inspect a source mutation without a browser | `controls.py` | Python 3.10+. |

Run commands below from the repository root. Use a new disposable evidence
workspace each time. Dependency installation may need npm registry access.
The control runner makes no model calls, external requests, or real uploads.

## Check integrity offline

```bash
python3 -B -m unittest discover -s tests/evals/pxp/upload-panel -p 'test_*.py' -v
```

Tests cover provenance, archive paths/links/size limits, exact mutations,
missing or ambiguous replacements, separation from the starter, and runner policy.
Tests that mock browser/process execution do **not** prove UI behavior.

## Run browser controls

Choose a new path in place of `run-NEW`:

```bash
python3 -B tests/evals/pxp/upload-panel/verify.py \
  "$PWD/.tmp/pxp-controls/run-NEW" --port 5191
```

Add `--include-alternatives` to check the two implementation alternatives as well:

```bash
python3 -B tests/evals/pxp/upload-panel/verify.py \
  "$PWD/.tmp/pxp-controls/alternatives-NEW" --include-alternatives
```

The runner:

1. Checks accepted hashes and creates five source trees, or seven with alternatives.
2. Installs locked dependencies with lifecycle scripts disabled. Variants share that
   install through local symlinks. This runner executes trusted control code only.
3. Runs accepted-app tests and coverage, plus alternative tests and coverage when
   selected. Builds every control so a build error cannot count as a detected defect.
4. Uses fresh browser sessions and loopback servers. Captures 436 × 406 at DPR 1,
   waits for fonts, and disables animations and carets.
5. Requires the accepted render to match its saved screenshot at threshold 0.
   Browser, OS, or font drift blocks the run; it is not permission to update the golden.
6. Runs the same content/Retry checks on every control. Compares each capture with
   both the design PNG and accepted render at threshold 8, without resizing,
   cropping, or ignored regions. Saves screenshots, overlays, reports, and JSON.
7. Closes its own browser/server and writes `results.json`.

| Exit | Meaning |
| --- | --- |
| `0` | Every control behaved as declared. |
| `1` | At least one control did not produce its expected checks. |
| `2` | Infrastructure error. |

Read logs as well as exit codes. Browser crashes and malformed results do not
count as detected defects. Failed workspace artifacts remain for diagnosis.

## Accepted reference

[`accepted/`](accepted/) contains a source-only app archive, design PNG, accepted
browser capture, original metrics/mask/overlay, capture settings, and checksummed
provenance. Cristian accepted the result with **“Looks good”**. The archive retains
the original app tests and six component tests, without installed dependencies,
transcripts, credentials, or browser profiles.

This is one acceptable implementation, **not the only correct implementation**.
Do not replace the blank starter in `skills/pxp/fixtures/` with it.
Keep the snapshot immutable; store a separately named example for a new acceptance.
Original metric paths are historical. `mask.png` and `overlay.png` are the retained
artifacts in this folder's accepted snapshot.

## Deliberate defects

[`variants.json`](variants.json) defines exact source replacements. Each variant
starts from the accepted archive, not from another modified variant. Full source
trees are created only in disposable workspaces.

| Control | Defect | Content check | Retry check | Initial image differs from accepted |
| --- | --- | --- | --- | --- |
| Accepted | None | Pass | Pass | No |
| Row spacing | 40px rows instead of 46px | Pass | Pass | Yes |
| Missing row | Completed video item omitted | Fail | Pass | Yes |
| Status colors | Completed text purple instead of green | Pass | Pass | Yes |
| Broken Retry | Failed item unchanged after Retry | Pass | Fail | **No** |

The browser checker does not receive a variant name or expected result. It checks
all seven visible names/states, exercises Retry, and captures the DOM. The runner
then compares observations with expectations. Detecting a known defect means the
**evaluator worked**, not that the component passed.

To inspect one mutation without a browser:

```bash
python3 -B tests/evals/pxp/upload-panel/controls.py \
  broken-retry "$PWD/.tmp/broken-retry-app"
```

The component regression test also catches broken Retry. After installing
dependencies, run this inside the accepted or broken app:

```bash
npm test -- src/UploadStatus.test.tsx -t 'Cancel all preserves'
```

Accepted passes; broken Retry produces an assertion failure, not a setup error.
The browser runner checks that transition independently.

## Acceptable alternatives and review status

[`alternatives.json`](alternatives.json) defines two small changes to the accepted
source, without modifying its snapshot:

| Alternative | Recorded evidence | Human review |
| --- | --- | --- |
| Grid rows | CSS Grid instead of Flexbox; pixel-identical at threshold 0 in the measured capture. | Pending. |
| Thinner Retry icon | 1.5px stroke; 113 changed pixels at threshold 8; same control name, hit area, content, and Retry behavior. | Accepted. |

Both passed all 22 app tests, coverage gates, build, and browser content/Retry
checks in the recorded run. They are variations of one implementation, not
independently built apps.

Cristian accepted the thinner icon with **“Yes”** to “Would you accept this thinner
icon too?”. [The review record](reviews/retry-icon.json) and
[reviewed screenshot](reviews/retry-icon.png) preserve that decision. It applies
only to this change, not all icon changes or a universal pixel tolerance.

Alternatives require passing content and Retry checks but do not prescribe
`visual_difference`. Some changed pixels alone do not reject them. Review the
runner's `retry-icon/accepted-delta-report.html` to inspect the change.

[`alternative-observations.json`](alternative-observations.json) stores measured
results and screenshot/checker hashes. [`observations.json`](observations.json)
is an earlier record for the original controls; its checker hashes refer to the
older code. Observation files retain their pre-review statuses. Use
`alternatives.json` and linked review records for current decisions.

## Verify one candidate independently

`ab_verify.py` checks a delivered app instead of trusting the agent's screenshots,
claims, or metrics. It expects the app directly at `outputs/app` within the run.
Run from a coordinator workspace that the candidate does not control:

```bash
python3 -B tests/evals/pxp/upload-panel/ab_verify.py \
  /absolute/run-directory /absolute/new-evidence-directory --port 5193
```

It installs locked dependencies, runs tests/build, and uses the Chromium settings
in `accepted/capture.json`. `ab_browser_check.js` captures raw transparent
`initial.png` and `final.png` from fresh seven-item state. It checks Cancel,
Cancel all, Retry, and keyboard collapse/expand separately.

The source must reproduce both raw captures at threshold 0 before reference
metrics are recorded. The coordinator produces source hashes, screenshots,
`reference-metrics.json`, mask, overlay, report, and `result.json`. Metrics are
review evidence, not an automatic visual-quality cutoff. Candidate source is
executable code: use a disposable environment without secrets or production access.

## Recorded experiments

These are individual experiments, not general claims about model or skill quality.
Keep failed runs and human feedback alongside metrics.

### First mid-tier pair: no completed comparison

[`mid-bundle-pilot.json`](mid-bundle-pilot.json) records one GPT-5.6 Luna/high pair:
skill + CLI versus neither, with the same neutral prompt and a 900-second cap.
The baseline finished but postprocessed screenshots and used five final items.
The assisted run timed out after setup and measurement work; its unfinished test
file failed the production type check. Both retained original tests and passed
18 tests plus coverage gates.

**No visual winner was established.** The record retains protocol, usage, artifact
hashes, and local evidence paths. It is a failed pilot, not a benchmark win.

### Revised mid-tier pair: metrics and human review disagreed

[`mid-bundle-v2.json`](mid-bundle-v2.json) records one completed GPT-5.6 Luna/high
pair. Both passed independent build/tests, seven-row content, Cancel/Cancel all/
Retry, keyboard collapse/expand, and raw capture reproduction.

Skill + CLI reduced measured changed ratio from 23.2747% to 17.8888% and perceptual
RMSE from 0.14251 to 0.13117. It used 2.23× wall time and 2.69× reported total tokens.

Blind review selected **Condition B**, the run with **neither skill nor CLI**, as
closer. [Exact feedback](mid-bundle-v2-feedback-blind.json) was “This looks closer”;
it did not accept either render. This disagreement establishes **no visual-quality
win** for skill + CLI. Investigate regional and alpha/shadow weighting before
another pair, and keep human review.

### DeepSeek 2×2 experiment: no delivered source

[`factorial-deepseek-v1.json`](factorial-deepseek-v1.json) records four skill/CLI
conditions with DeepSeek v4 Flash/high. Preflight confirmed CLI access only in
its two intended conditions and rejected explicit browser selection everywhere.
All four timed out at 900 seconds before delivering source. There is no functional
or visual result from which to infer an effect. Do not repeat the setup unchanged.

## What the controls prove

The recorded control run reproduced the accepted image exactly and detected all
four deliberate defects. Broken Retry had **zero visual difference**. Screenshots
therefore cannot replace behavior checks. The wrong status color changed the
whole-image ratio only slightly, despite being a deliberate defect.

`visual_difference` means different from the accepted control, not unacceptable
to a human. Exact reproduction checks the environment for this mutation experiment;
it does not require future implementations to use the same source or pixels.

These controls show detection of four known defects. They do not establish that
other acceptable implementations will pass. Do not turn the accepted design
difference of 13.36% into a universal cutoff.

Next, complete human review of the Grid alternative and gather more acceptable
implementations and small defects. Keep human visual acceptance separate from
behavior/content checks until proposed region-specific gates have been calibrated
and validated.
