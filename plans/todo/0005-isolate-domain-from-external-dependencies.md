---
id: TASK-0005
title: Isolate domain logic from all external dependencies
status: doing
depends_on: [TASK-0002,TASK-0004]
priority: normal
tags: []
---

# Isolate domain logic from all external dependencies

## Problem
Domain logic must not depend on external mechanisms. Current image APIs combine computation with filesystem operations, and other boundaries need an explicit audit for framework, provider, configuration and serialization coupling. The goal is Clean Architecture dependency direction, not only I/O extraction.

## Evidence
- `internal/imagediff/decoded_images.go`: `LoadDecodedImages` loads a pair from paths.
- `internal/imagediff/image.go`: `DecodedImages.Compare` accepts a mask output path; PNG decoding, encoding and cropping mix filesystem operations with analysis.
- `internal/imagediff/overlay.go`: `WriteOverlay` computes and persists an overlay.
- `internal/imagecontext/client.go` already defines a small `Client.Describe(context.Context, Input)` interface and provider factory. Review its use before introducing any further provider abstraction.

## Dependencies
Build on TASK-0002's in-memory probe/scan API and TASK-0004's artifact owner rather than create competing image and file boundaries. Package renaming (TASK-0003) is independent; use whichever names exist when implementation starts.

## Architecture target
Dependencies point inward: entrypoints and adapters depend on application contracts and domain logic, never the reverse.
- Domain owns deterministic image evidence, annotation geometry and domain validation. It has no third-party imports or dependencies on Cobra, HTTP, providers, environment/configuration, filesystem, clocks/randomness, process globals, serialization or presentation packages.
- Pure standard-library value types and algorithms (such as `image.Image`, image/color, math and errors) are allowed. Dependency-free means independent of external mechanisms, not reimplementing Go primitives.
- Application workflows coordinate domain operations through explicit dependencies. Framework-specific flags/errors, provider DTOs, serialized envelopes and file paths used for persistence stay at outer boundaries.
- Adapters own external mechanisms: filesystem/codecs, HTTP/provider protocols, configuration loading, JSON/TOON/CSV, HTML and process integration.
- Composition owns concrete construction and environment access. Consumer-owned ports exist only at actual substitution boundaries; use functions and existing Go interfaces where sufficient.

## Approach
0. Inventory all domain/application imports and side effects, not only image I/O. Inspect annotations, comparison, inspection, validation and advisory-context orchestration. Document each dependency's owner and desired direction, including third-party libraries, config/global access, transport types and serialization metadata. Move domain data out of adapter-owned packages when necessary; do not invert imports by creating a shared grab bag. Choose package changes from this inventory rather than impose a generic layered tree.
1. Read module guides and inspect semantic callers of path-based image APIs. Record existing file ownership, decoding/normalization rules, errors and artifact timing before choosing signatures.
2. Add characterization tests before extraction. Keep threshold, alpha, mask, crop and overlay semantics unchanged.
3. Expose deterministic computation over decoded images and explicit options. Return measurements and generated images; do not create files inside computation. Avoid constructing an abstraction for every function or struct.
4. Put image file loading and persistence at a narrow adapter boundary. Prefer `io.Reader`/`io.Writer` for codecs and decoded `image.Image` or existing concrete image types for algorithms. Keep NRGBA normalization explicit and behavior-compatible. Start with the artifact package from TASK-0004 if cohesive; use a separate image-codec package only if inspection establishes a distinct responsibility.
5. Wire adapters at command/executable composition boundaries. Keep opening/closing files, temporary-directory cleanup, environment reads, configuration and exit policy explicit at the edge.
6. Review provider configuration/client construction and request wiring. Reuse the existing small interface if sufficient; define any new interface at its consumer only when a concrete need is demonstrated. Move environment-dependent configuration to composition if needed. Do not create a central interfaces folder.
7. Migrate command callers. Remove superseded internal path wrappers once callers migrate, or document any intentional compatibility adapter outside the computation package.

## Acceptance criteria
- [ ] Document the domain/application/adapter/composition package map and the full external-dependency inventory, with every dependency assigned an owner.
- [ ] Domain packages have no third-party or outward imports and no mechanism side effects. Domain tests run offline without adapter initialization, filesystem fixtures or environment setup.
- [ ] Application orchestration depends on domain and consumer-owned ports, not concrete providers, Cobra, filesystem helpers or serializers. Composition supplies adapters explicitly; no service locator or global singleton.
- [ ] Boundary mappings keep provider/framework/output contracts out of domain APIs. Preserve public JSON/TOON/CSV contracts using edge DTOs where required; remove serialization-specific tags from domain types when moving those contracts.
- [ ] Add a deterministic architecture test using Go import/package metadata to reject forbidden domain imports and application-to-concrete-adapter dependencies. Include positive and negative tests of the rule so future coupling fails verification.
- [ ] Compare, probe, scan, crop and overlay computation can be exercised with in-memory images and no filesystem, environment or network dependency.
- [ ] Deterministic computation neither imports artifact/CLI/provider packages nor opens, creates or writes files; adapters own those operations.
- [ ] Core tests cover identical/changed images, dimensions, thresholds, alpha semantics, excluded pixels, invalid bounds, masks and overlays with deterministic fixtures.
- [ ] Adapter tests cover malformed PNGs, missing files, write failures, directory creation, permissions and cleanup. Use small explicit fakes or standard readers/writers rather than a generic filesystem interface.
- [ ] Existing command and smoke tests preserve JSON/TOON/CSV contracts, image pixels, artifact paths, error classifications, exit codes and relevant failure/cleanup behavior.
- [ ] Provider review records whether existing injection is sufficient. Any wiring changes use mocked requests and cover successful descriptions and provider failures; no paid calls or new provider features.
- [ ] Capture red/green evidence, focused coverage and caller/test impact. Obtain fresh watcher verification using configured targets before completion.
- [ ] Update module guides to identify computation, adapter and composition ownership. No second generic `infra/io` or `utils` owner is introduced.

## Non-goals
No decode-caching/performance project, changed color math, atomic-write feature, new provider, universal filesystem interface or ceremonial layer/interface for every type. A focused domain/application/adapter separation is in scope; generic infrastructure hierarchies without a concrete owner are not. Preserve evaluation input/grader separation.

## Delivery
Start with the dependency inventory and package-boundary design. Split implementation into additional dependent board tasks if that inventory makes this too large for reviewable commits. Suggested slices: (1) characterize and isolate domain computation/data, (2) isolate application workflows and ports, (3) migrate concrete adapters and composition, (4) enforce dependency rules and verify contracts. Review all external mechanisms; reuse existing provider contracts where sufficient rather than add duplicate abstractions. Each implementation commit references its task ID. Move TASK-0005 to `plans/done` only when the complete dependency-direction acceptance gate passes.
