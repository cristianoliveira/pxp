---
id: TASK-0021
title: Prepare v0.2.0 README and release communication
status: done
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
- [x] Review the existing release workflow, version injection, tags, and release-note conventions; record the exact preparation and publication commands.
- [x] Rewrite README communication in STE100/Minto style: lead with current comparison and `pxp review` workflows, explain machine-readable output versus blocking human review, and remove stale or ambiguous examples.
- [x] Document v0.2.0 capabilities and the breaking `--report` migration accurately, without claiming human approval or unsupported behavior.
- [x] Add/update release notes for v0.2.0, including migration guidance and notable quality/accessibility improvements.
- [x] Have QA independently verify every README command/example, help text, release-note claim, and release build/checksum workflow against current main.
- [x] Prepare a release-ready PR and exact tag/release command; do not create or push `v0.2.0` or publish a GitHub release until explicit confirmation.

## Completion evidence
README and release preparation merged in PR #41 at `52fb4ad`. Release version injection fixes merged in PR #42 at `fd03559`. Kelly independently verified README/help commands, migration guidance, Nix build/run, version injection, six cross-platform archives, checksums, workflow behavior, and quality/regression evidence.

After separate explicit publication authorization, annotated tag `v0.2.0` was created at verified `origin/main` `fd03559`. Release workflow `34864265787` passed and published six archives plus checksums. All six checksums verified. GitHub Release: https://github.com/cristianoliveira/pxp/releases/tag/v0.2.0.

## Notes
The release workflow runs on `v*` tags, tests/builds linux/darwin/windows for amd64/arm64, and publishes GitHub Release assets. Preparation remained separate from publication until the user explicitly authorized the tag.

