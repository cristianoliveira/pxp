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
- [x] Reviewed the release workflow, version injection, tags, and release-note conventions; recorded exact preparation and publication commands.
- [x] Rewrote README communication in STE100/Minto style around comparison and `pxp review`, machine-readable output, blocking human review, and current examples.
- [x] Documented v0.2.0 capabilities and the breaking `--report` migration without claiming human approval or unsupported behavior.
- [x] Added v0.2.0 release notes with migration guidance and quality/accessibility improvements.
- [x] QA independently verified README commands/examples, help text, release-note claims, and release build/checksum workflow against current main.
- [x] Prepared release-ready PR #41 and exact tag/release commands; no `v0.2.0` tag or GitHub release was created.

## Notes
Current release workflow: `.github/workflows/release.yml` runs on `v*` tags, tests/builds Linux/macOS/Windows for `amd64`/`arm64`, injects the tag into binaries, and publishes GitHub Release assets with checksums. Current tag: `v0.1.0`. This task prepares communication and release metadata only; publication remains a separate explicit action.

## Completion evidence

- PR #41 merged to `origin/main` at `52fb4ad`; final reviewed commit was `0088724`.
- Kelly independently PASSed the final commit. Report: `.tmp/reports/14-09-26/task-0021-qa.md`.
- Watcher generation 40 passed format, vet, lint, full tests, review browser e2e, and install.
- Quality passed at 79.6% coverage; `make quality-test`, race tests, and diff checks passed.
- `nix build .#pxp --no-link` and `nix run .#pxp -- --version` passed with `pxp version 0.2.0`.
- Release-equivalent Linux/macOS/Windows `amd64`/`arm64` archives, bundled README files, six checksums, and v0.2.0 binary version injection were verified.
- Compatibility policy: the `--report` removal is a deliberate breaking change. Legacy use exits with a migration diagnostic to `pxp review`; it is not silently ignored.
- No `v0.2.0` tag was created or pushed. No GitHub release was published. Only `v0.1.0` exists.
- Board closed from clean latest `origin/main` (`52fb4ad`) with the transition commit below.

