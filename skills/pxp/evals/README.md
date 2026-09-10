# pxp skill evaluations

Measure whether the skill helps an agent build a working UI from a PNG. Knowing
CLI flags is not enough: the result must have real source, working controls, and
independently checked screenshots.

Cases and expectations are in [`evals.json`](evals.json), in skill-creator format.
No external service or live upload is needed. Model runs can use paid provider
calls; agree on scope before starting them.

Follow this order: check fixtures, agree on run limits, run candidate and baseline,
then grade retained evidence. Keep accepted solutions out of candidate inputs.

## Cases and success criteria

1. **Reusable upload panel:** build real UI from the PNG; implement accessible
   local cancel/retry/collapse actions; demonstrate measured visual improvement.
2. **App integration:** add the same component to the existing workspace;
   preserve task/attachment behavior and prove both isolated and integrated use.
3. **Larger capture, tight budget:** keep 900 × 700 screenshots; compare the
   component using explicit `--actual-crop 32,24,436,406`; stop after at most two
   refinement iterations and report remaining differences honestly.

Each case gets the same source archive, reference PNG, and fixed data. Do not
pre-implement the target component in the fixture. See
[fixture instructions](../fixtures/README.md) for dependencies and repacking.

## 1. Check inputs offline

From the repository root:

```sh
python3 -B -m unittest discover -s skills/pxp/evals -p 'test_*.py' -v
```

These checks validate packaging, fixture integrity, required fields, and crop
geometry. They do **not** measure model quality. In a disposable extracted app,
run `npm ci --ignore-scripts`, `npm run test:coverage`, and `npm run build` before
spending model calls. See the [fixture guide](../fixtures/README.md) for setup.

## 2. Agree on scope and run a comparison

Choose the model/tier, cases, repetitions, concurrency, timeouts, and refinement
limits before running. A small first check is **case 1, one candidate/baseline
pair, sequential, 900 seconds per run, five refinement iterations**.

That is two model executions; grading calls cost extra. One pair checks setup,
not reliability. A larger run with all three cases and three repetitions per side
requires 18 executions. Case 3 keeps its stricter two-iteration limit.

Use the installed `skill-creator` runner; do not add another evaluation engine.
Save the complete original skill outside the candidate before edits. If that
snapshot contains installed dependencies, make a separate baseline execution copy
with its unchanged `SKILL.md` and required references/scripts only. Supply the same
fixtures through `--file` to both sides and record the preparation in run metadata.
Never include `node_modules` or old session transcripts as skill content.

Example setup: start at the repository root. Replace the absolute paths below
with your installed runner, saved baseline, and a new iteration directory. Read
the installed runner's help before use; its interface can change.

```sh
export SKILL_ROOT="$PWD/skills/pxp"
export CREATOR="/absolute/path/to/skill-creator"
export BASELINE_SKILL="/absolute/path/to/pre-change-execution-copy"
export CASE_ROOT="/absolute/path/to/pxp-workspace/iteration-1/eval-1-upload-panel"
python3 - <<'PY'
import json, os
from pathlib import Path
root = Path(os.environ['CASE_ROOT'])
root.mkdir(parents=True, exist_ok=False)
case = json.loads((Path(os.environ['SKILL_ROOT']) / 'evals/evals.json').read_text())['evals'][0]
(root / 'prompt.txt').write_text(case['prompt'])
(root / 'eval_metadata.json').write_text(json.dumps({
    'eval_id': case['id'], 'eval_name': 'upload-panel',
    'prompt': case['prompt'], 'assertions': case['expectations'],
    'baseline_skill': os.environ['BASELINE_SKILL'],
}, indent=2) + '\n')
PY

cd "$CREATOR"
python3 -m scripts.run_case \
  --prompt-file "$CASE_ROOT/prompt.txt" \
  --skill-path "$SKILL_ROOT" \
  --file "$SKILL_ROOT/fixtures/todoapp.tar.gz" \
  --file "$SKILL_ROOT/fixtures/upload-modal-multifiles.png" \
  --file "$SKILL_ROOT/fixtures/upload-state.json" \
  --run-dir "$CASE_ROOT/with_skill/run-1" --tier high --timeout 900

python3 -m scripts.run_case \
  --prompt-file "$CASE_ROOT/prompt.txt" \
  --skill-path "$BASELINE_SKILL" \
  --file "$SKILL_ROOT/fixtures/todoapp.tar.gz" \
  --file "$SKILL_ROOT/fixtures/upload-modal-multifiles.png" \
  --file "$SKILL_ROOT/fixtures/upload-state.json" \
  --run-dir "$CASE_ROOT/without_skill/run-1" --tier high --timeout 900
```

`--tier high` is illustrative; use the agreed, configured tier identically on
both sides. The `without_skill` directory holds the old-skill baseline here,
not a no-skill run. The runner copies inputs under `inputs/` and asks for
outputs under `outputs/`. It isolates context, **not the host filesystem or
network**. Use disposable workspaces, loopback servers, and no production data.
Stop browsers/dev servers after each run; never reuse a run directory.

## 3. Grade evidence, not the agent's claims

Follow skill-creator's `agents/grader.md`. Save `grading.json` beside each run's
`timing.json` with expectation `text`, `passed`, and specific `evidence`, plus
summary counts and pass rate. Review these gates:

- Run delivered source/build/tests; inspect controls in a browser. Test-file
  presence alone does not prove behavior. Check the original tests were retained.
- Check screenshots against the actual reference and source. Require a real DOM
  implementation, not the PNG in an `<img>`, canvas, or traced substitute.
- Re-run recorded comparisons from retained images with the same CLI version and
  options. Compare emitted JSON to reported values. A mask filename is not proof.
- Inspect transcript/capture commands for baseline timing, fixed viewport/device
  scale, stable state, unchanged thresholds/crops, and bounded iteration count.
- Assess visual resemblance and remaining differences by human review. Metric
  improvement alone can reward a poor first render; it is not an absolute quality
  gate. Do not invent an uncalibrated visual pass threshold after seeing results.
- Count runtime/provider/tool setup failures as infrastructure errors, not
  successful negative cases. Absence of metrics is not a verified match.

Use a separate grader where practical. If the same agent grades its own work,
state that limit. Keep expectations fixed for both sides. If a check is flawed,
record the change and regrade both sides. Report pass rates, time, tokens, and
variation across runs.

From skill-creator, aggregate and generate its existing review viewer:

```sh
python3 -m scripts.aggregate_benchmark "$(dirname "$CASE_ROOT")" --skill-name pxp
python3 eval-viewer/generate_review.py "$(dirname "$CASE_ROOT")" \
  --skill-name pxp \
  --benchmark "$(dirname "$CASE_ROOT")/benchmark.json" \
  --static "$(dirname "$CASE_ROOT")/review.html"
```

## 4. Keep results separate from inputs

Keep screenshots, agent outputs, reports, and run workspaces outside the skill
and untracked. Stop each run's browser and server before starting another run.
Never reuse a run directory or copy an accepted solution into the starter fixture.

These cases test execution with a supplied skill. They do not test whether an
agent selects that skill from its installed catalog. A successful run also does
not establish a universal pixel tolerance or a general quality improvement.
