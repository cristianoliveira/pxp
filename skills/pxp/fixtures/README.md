# Component implementation fixtures

These files are inputs for the [pxp skill evaluations](../evals/README.md).
They do not contain a completed upload status panel. The evaluated agent must
build it.

| File | Purpose |
| --- | --- |
| `todoapp.tar.gz` | Source-only React/TypeScript/Vite app with original tests and an npm lockfile. |
| `upload-modal-multifiles.png` | Unmodified 436 × 406 reference, including shadow padding. |
| `reference.sha256` | Reference checksum to verify before and after a run. |
| `upload-state.json` | Fixed seven-row state: two uploading, one failed, and four completed. |

The PNG defines appearance; JSON supplies content and state. The Shared Drive
label is not a request for a remote upload service. Use local actions and fixed
progress values, not timers. Exact source font metadata is unavailable; report
remaining typography differences rather than dismissing them.

## Requirements

- Python 3.10+ for fixture tools.
- Node.js 22.12+ and npm 11 for the app.
- Browser capture tooling and `pxp` for implementation evaluations.
- npm registry access when locked dependencies are not cached.

No external service or live upload is needed.

## Unpack and check

Run from this fixtures directory. `mktemp` creates a new workspace for each run:

```bash
WORKSPACE="$(mktemp -d "${TMPDIR:-/tmp}/pxp-fixture.XXXXXX")"
tar -xzf todoapp.tar.gz -C "$WORKSPACE"
cd "$WORKSPACE/todoapp"
npm ci --ignore-scripts
npm test
npm run build
npm run dev -- --host 127.0.0.1 --port 5173 --strictPort
```

The final command starts a foreground server; stop it when finished. For concurrent
runs, select a different port and browser session for each workspace. Keep servers
on loopback.

The archive preserves the original app README and tests. The normal homepage uses
browser storage and date-relative sample tasks. For panel captures, use an isolated
preview of the new component with `upload-state.json` and a fresh browser context.
Do not use the homepage's changing seed data as screenshot state.

## Check fixture integrity

From the repository root:

```bash
python3 -B -m unittest discover -s skills/pxp/evals -p 'test_*.py' -v
```

These offline tests check the checksum, fixed state, archive contents, and evaluation
inputs. Passing them does not prove that a model can implement the component.

## Update the archive

Edit an unpacked source copy and verify its tests and build. Then, from the
repository root, replace the path below with that copy:

```bash
python3 skills/pxp/evals/pack_fixture.py \
  /absolute/unpacked/todoapp skills/pxp/fixtures/todoapp.tar.gz
python3 -B -m unittest discover -s skills/pxp/evals -p 'test_*.py' -v
```

The packer:

- Allows selected source, configuration, and asset files, including the lockfile,
  `.npmrc`, `.gitignore`, and tests.
- Excludes dependencies, build and coverage output, caches, browser artifacts,
  and agent histories.
- Rejects symlinks and normalizes archive metadata.
- Produces identical bytes for identical source on the same Python/zlib toolchain.

Review the allowlist before adding a file type. Keep only the source archive here,
not a second expanded app. The evaluation runner copies the whole skill into each
candidate run, so dependencies and generated output would add unnecessary data.

Never replace the starter with an accepted solution. Accepted implementations and
grader evidence stay outside `skills/` and must not be passed to candidate agents.
A source-only archive saves copy space; each disposable run still needs dependencies.
