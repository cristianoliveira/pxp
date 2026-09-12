# Purpose

`internal/review` owns short-lived, localhost-only annotated review rounds: immutable comparison snapshots, embedded frontend serving, and explicit feedback persistence.

# Boundaries

It serves a review protocol and preserves pixel-coordinate evidence. It does not own Cobra command policy, image algorithms, generic artifact primitives, or provider descriptions.

# Connections

- [Command orchestration](internal/commands/AGENTS.md): starts and coordinates review rounds from CLI options.
- [Image comparison](internal/imagediff/AGENTS.md): provides deterministic comparison and overlay data for a snapshot.
- [Image I/O](internal/imageio/AGENTS.md): decodes source images and writes PNG snapshot artifacts.
- [Artifact persistence](internal/artifact/AGENTS.md): provides filesystem creation used while materializing the round.

# Landmarks

- `internal/review/review.go:NewSession`: creates an immutable review snapshot and feedback lifecycle.
- `internal/review/review.go:Session.Submit`: validates and records an explicit review decision.
- `internal/review/server.go:NewServer`: exposes the review session over a loopback HTTP server.
- `internal/review/server.go:Server.WaitContext`: waits for feedback or safely cancels the round.

# Boundary flows

- Information flow: `internal/review/review.go:NewSession` -> `internal/review/server.go:NewServer` via `internal/commands/command.go:NewCommand`; value: `*review.Session`.
- Information flow: `internal/review/review.go:Session.Submit` -> `internal/commands/command.go:NewCommand` via `internal/review/server.go:Server.WaitContext`; value: `review.Result`.

# Placement

Keep review protocol, snapshot integrity, and browser transport together. Put reusable image evidence in `imagediff`, file codecs in `imageio`, and command sequencing in `commands`.
