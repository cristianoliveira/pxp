---
id: TASK-0021
title: Prepare v0.2.0 README and release communication
status: doing
depends_on: []
priority: normal
tags: []
---

# Prepare v0.2.0 README and release communication

## Problem
The project is currently tagged v0.1.0 while main contains the review-loop workflow, quality guardrails, complexity refactor, compact accessible annotation markers, and removal of static --report review output. Prepare accurate user-facing communication and release metadata for v0.2.0 without publishing the tag or GitHub release until explicitly confirmed.

## Context
The repository has a single published tag, `v0.1.0`, while `main` now contains the local annotated `pxp review` workflow, quality guardrails, complexity improvements, accessible numbered annotation markers, and the breaking removal of comparison `--report`. The README and release notes must describe the current product accurately before preparing `v0.2.0`.

## Acceptance criteria
- [ ] Review the existing release workflow, version injection, tags, and release-note conventions; record the exact preparation and publication commands.
- [ ] Rewrite README communication in STE100/Minto style: lead with current comparison and `pxp review` workflows, explain machine-readable output versus blocking human review, and remove stale or ambiguous examples.
- [ ] Document v0.2.0 capabilities and the breaking `--report` migration accurately, without claiming human approval or unsupported behavior.
- [ ] Add/update release notes for v0.2.0, including migration guidance and notable quality/accessibility improvements.
- [ ] Have QA independently verify every README command/example, help text, release-note claim, and release build/checksum workflow against current main.
- [ ] Prepare a release-ready PR and exact tag/release command; do not create or push `v0.2.0` or publish a GitHub release until explicit confirmation.

## Notes
Current release workflow: `.github/workflows/release.yml` runs on `v*` tags, tests/builds linux/darwin/windows for amd64/arm64, and publishes GitHub Release assets. Current tag: `v0.1.0`. This task prepares communication and release metadata only; publication remains a separate explicit action.

