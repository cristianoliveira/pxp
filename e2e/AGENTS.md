# Purpose

`e2e/` owns browser-level checks for the review workflow exposed by [the review capability](internal/review/AGENTS.md).

# Boundaries

These checks exercise the packaged CLI and browser protocol from outside the runtime packages. They do not own review behavior or production fixtures.

# Connections

- [Review](internal/review/AGENTS.md): provides the localhost review server and browser contract under test.
- [Command orchestration](internal/commands/AGENTS.md): provides the CLI entrypoint used to launch review workflows.

# Placement

Put end-to-end browser checks here when they validate a cross-package user flow. Keep unit tests beside the runtime package they isolate.
