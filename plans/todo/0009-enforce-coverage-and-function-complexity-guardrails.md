---
id: TASK-0009
title: Enforce coverage and function complexity guardrails
status: todo
depends_on: []
priority: normal
tags: [quality, coverage, complexity]
---

# Enforce coverage and function complexity guardrails

## Problem
Coverage is collected but has no minimum threshold, and function complexity is not checked by linting, so regressions can pass all current local and CI gates without detection.

## Context
The current CI workflow uploads Go coverage but does not fail on coverage loss. The watcher and hooks run linting, but `.golangci.yml` does not enable a complexity or function-length check. Complexity hotspots already exist, so the policy must distinguish an agreed baseline from new regressions.

## Acceptance criteria
- [ ] Define and document minimum coverage and maximum function-complexity/function-length thresholds, including scope, exclusions, and baseline handling.
- [ ] Provide one canonical local command that evaluates both policies and exits non-zero when either threshold is violated.
- [ ] Wire the canonical command into CI and document the equivalent local invocation; avoid a CI-only check.
- [ ] Decide explicitly whether coverage and complexity belong in the normal watcher/pre-commit gate or in an advanced quality gate, and keep the lifecycle commands consistent with that decision.
- [ ] Add deterministic boundary tests or fixtures proving passing and failing coverage/complexity cases.
- [ ] Provide a runnable browser-test harness for `e2e/review_browser_check.js` and have the watcher execute the behavioral e2e checks, not only syntax validation.
- [ ] Record the initial baseline and identify any existing hotspots that require remediation or an explicit temporary exemption.

## Notes
Do not silently lower thresholds to accommodate existing code. Prefer a diff/new-code policy when enforcing a whole-repository threshold would block unrelated work.

The watcher currently validates the committed e2e helper's JavaScript syntax with `node --check`; it does not yet run browser behavior because the repository has no pinned Playwright runner or stable review fixture URL. Replace this interim check when the harness is added.

