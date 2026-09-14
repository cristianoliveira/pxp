# Anatomy: element discovery from a single image

## Vision
Make pxp answer "what is in this screenshot, and where" from one image — deterministically, offline, without a reference.

Today every pxp answer is comparative: it needs two equal-sized PNGs. Agents and humans often hold only one image and still need absolute facts: where the text lines are, how big the icon is, what the dominant ink colors are. During the card-alert implementation (2026-09) this gap forced a throwaway Go helper and a dozen single-line scans to relocate layout before any comparison could help.

This document defines desired outcomes, not an implementation recipe. It records the direction agreed with the user; it does not claim the current CLI offers the capability.

## Background: why compare cannot answer this today
`findRegions` (internal/imagediff/image.go) already discovers true connected components of changed pixels. The gap is upstream of it: compare requires a reference, and the blank-canvas workaround (diffing against a synthetic white PNG) collapses into one region because `groupImageRegions` (internal/commands/compare.go) merges regions whose bounds sit within `region-gap` — a full-canvas frame region swallows every interior element. That grouping is correct diff-triage behavior; anatomy needs the opposite default.

## Goals

### 1. Element discovery needs only one image
`pxp anatomy <image.png>` reports element bounds from a single file. Background is detected from the image itself (most frequent image color, deterministic tie-break; explicit `--background #RRGGBB` overrides).
 Content is pixels differing from background by more than the comparison threshold.

Success means the card-alert fixture yields its known elements (icon, title band, metadata band, three description lines, two buttons, ellipsis) as separate regions without any companion file.

### 2. Grouping is explicit, never surprising
Raw connected components are the default. Merging nearby components happens only when the caller asks for it (`--group N`), so a three-line paragraph can become one block on demand — and never merges silently.

Success means running anatomy twice with and without `--group` changes only the grouping, and the same invocation is byte-for-byte deterministic.

### 3. Output is agent-first and bounded
Default structured output follows the CLI's existing contract (TOON by default, `--json` compatibility, `--format` selection), with `total`, `returned`, and `truncated` fields and a `--full` escape hatch. Default fields are the decision-relevant few: bounds, pixels, density, dominant color. Elements sort in reading order (top-to-bottom, left-to-right).

Success means an agent can decide "what is this image's layout and where do I scan next" from one unbounded-until-truncated response, with copy-pasteable follow-up commands.

### 4. Boundaries stay clean
Element-discovery evidence lives in `internal/imagediff` (reusing and, where needed, extracting shared connected-component helpers); command policy lives in `internal/commands`; presentation flows through `internal/output`. The user-facing contract is documented in `docs/command-contracts.md`.

Success means no presentation logic in imagediff and no image traversal in commands.

### 5. Non-goals
Anatomy does not label semantics ("this is a button"), does not OCR, does not call providers, and does not compare two images. Semantic naming would require models and would break pxp's deterministic-evidence contract.

## Test surface (per AXI)
Happy path against the card-alert fixture; explicit empty result on a solid-color image (silence must not look like failure); threshold extremes; `--min-pixels` speck filtering; truncation with `--full` escape; `--background` override; usage errors for invalid formats and bounds; determinism (same input, same bytes).
