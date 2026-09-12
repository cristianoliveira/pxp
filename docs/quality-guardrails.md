# Quality guardrails

`make quality` is the canonical advanced quality gate. It is required in CI and
is the local command to run before requesting review. It exits non-zero when
coverage or complexity policy fails.

## Coverage policy

- **Minimum:** 78.0% statement coverage.
- **Scope:** `./internal/...` and `./tests/smoke`.
- **Excluded:** `cmd/pxp` has no testable statements and is excluded from the
  profile; generated code is excluded.
- **Baseline:** the initial profile measured 78.3%. The 0.3-point margin is
  intentional so a harmless package/test split does not create a noisy gate.
  The threshold must not be lowered to accommodate a regression.
- **Command:** `scripts/quality-gate.sh` runs the scoped tests and parses
  `go tool cover -func` output. It writes `coverage.out` for CI upload.

## Complexity policy

`golangci-lint` enforces these repository-wide limits for non-generated Go
files, including tests:

- Cyclomatic complexity: **40** per function.
- Function length: **203** lines or **127** statements.

The limits are the initial baseline maxima, not style targets. Existing
hotspots remain explicit baseline exceptions until separately refactored:

- `internal/commands/compare.go:runComparisonCommand` (complexity 40, 203
  lines).
- `internal/review/review.go:NewSessionWithContext` (complexity 33, 164
  lines, 94 statements).
- `internal/imagediff/image.go:DecodedImages.Compare` (complexity 31, 148
  lines, 87 statements).
- `internal/review/review.go:validateRequest` (complexity 30).

New code must stay within the limits. Lowering the limits or adding an
exclusion requires a separate, documented refactoring decision.

## Lifecycle choice

Coverage and complexity are an **advanced quality gate**, not part of the
normal watcher or pre-commit hooks. The watcher and hooks remain fast for edit
feedback; CI and `make quality` provide the authoritative policy check.

`make quality-test` runs deterministic boundary fixtures: a passing and failing
synthetic coverage profile, a function over the complexity limit, and a
function over the length limit.

## Browser behavior gate

The watcher runs `scripts/run-review-browser-check.sh`, which starts a fresh
local review server and executes `e2e/review_browser_check.js` through
`playwright-cli` 0.1.9 and Chromium. The helper intercepts final feedback, so
it exercises behavior without submitting a live review. The script cleans up
the browser and server and can preserve proof artifacts with
`PXP_E2E_PROOF_DIR=/path/to/proof`.
