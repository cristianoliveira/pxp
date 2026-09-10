# Domain boundary inventory

- `annotations`: pure coordinate validation/intersections; no persistence or serialization imports.
- `imagediff`: pure decoded-image metrics and generated in-memory images; no codec, filesystem, CLI, provider, or presentation imports.
- `annotationio`: JSON wire DTOs and file persistence; delegates invariants to `annotations` and file creation to `artifact`.
- `imageio`: PNG decoding/normalization and mask, crop, and overlay persistence; delegates computation to `imagediff`.
- `commands`: Cobra workflow/composition boundary; loads adapters and supplies paths/options to domain operations.
- `artifact`: filesystem creation and parent-directory ownership.
- `report`, `output`: HTML and structured serialization at presentation edges.
- `imagecontext`: provider HTTP/configuration at the optional integration edge.
- `cli`, `cmd/pxp`: process, exit policy, and executable composition.

The architecture test rejects persistence/codec imports from domain packages and asserts adapter wiring at the command boundary. Filesystem permission and close behavior remain platform-dependent adapter contracts; deterministic missing-path and creation-failure cases are tested. JSON tags on `imagediff` result structs remain the existing structured-output contract and require a coordinated output DTO migration, explicitly deferred from TASK-0005.
