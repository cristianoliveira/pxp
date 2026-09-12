#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
SESSION=${PXP_PLAYWRIGHT_SESSION:-pxp-review}
PROOF_DIR=${PXP_E2E_PROOF_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/pxp-review-e2e.XXXXXX")}
PID=''

cleanup() {
  playwright-cli -s="$SESSION" close >/dev/null 2>&1 || true
  if [[ -n "$PID" ]] && kill -0 "$PID" 2>/dev/null; then
    kill "$PID" 2>/dev/null || true
    wait "$PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

if ! command -v playwright-cli >/dev/null 2>&1; then
  echo 'e2e: playwright-cli 0.1.9 is required' >&2
  exit 1
fi
if [[ "$(playwright-cli --version)" != '0.1.9' ]]; then
  echo "e2e: expected playwright-cli 0.1.9, found $(playwright-cli --version)" >&2
  exit 1
fi

mkdir -p "$PROOF_DIR"
cp "$ROOT_DIR/tests/smoke/fixtures/image-diff/real-ui-reference.png" "$PROOF_DIR/reference.png"
cp "$ROOT_DIR/tests/smoke/fixtures/image-diff/real-ui-implementation.png" "$PROOF_DIR/actual.png"
go build -o "$PROOF_DIR/pxp" ./cmd/pxp
"$PROOF_DIR/pxp" review "$PROOF_DIR/reference.png" "$PROOF_DIR/actual.png" \
  --out "$PROOF_DIR/review" --json >"$PROOF_DIR/stdout" 2>"$PROOF_DIR/stderr" &
PID=$!

for _ in $(seq 1 100); do
  if url=$(awk '/pxp review listening at/{print $5}' "$PROOF_DIR/stderr"); [[ -n "$url" ]]; then
    break
  fi
  if ! kill -0 "$PID" 2>/dev/null; then
    cat "$PROOF_DIR/stderr" >&2
    exit 1
  fi
  sleep .1
done
if [[ -z "${url:-}" ]]; then
  echo 'e2e: review server did not publish a URL' >&2
  cat "$PROOF_DIR/stderr" >&2
  exit 1
fi

playwright-cli -s="$SESSION" open "$url" --browser chrome >/dev/null
runner="$PROOF_DIR/browser-runner.js"
ROOT_DIR="$ROOT_DIR" RUNNER="$runner" node <<'NODE'
const fs = require('node:fs');
const source = fs.readFileSync(`${process.env.ROOT_DIR}/e2e/review_browser_check.js`, 'utf8')
  .replace("const assert = require('node:assert/strict');", `const assert = {
    equal(actual, expected, message) { if (actual !== expected) throw new Error(message || ('expected ' + expected + ', got ' + actual)); },
    ok(value, message) { if (!value) throw new Error(message || 'expected a truthy value'); },
    match(value, pattern, message) { if (!pattern.test(value)) throw new Error(message || ('expected ' + value + ' to match ' + pattern)); },
    doesNotMatch(value, pattern, message) { if (pattern.test(value)) throw new Error(message || ('expected ' + value + ' not to match ' + pattern)); },
  };`)
  .replace('module.exports = {runReviewBrowserChecks};', 'return runReviewBrowserChecks(page, {url: page.url()});');
fs.writeFileSync(process.env.RUNNER, `(async (page) => {\n${source}\n})`);
NODE
playwright-cli -s="$SESSION" run-code --filename "$runner"
