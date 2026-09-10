# Purpose

`internal/imagecontext` provides optional OpenRouter and OpenAI visual descriptions for comparison regions. It is advisory context layered on top of deterministic image evidence.

# Boundaries

The package owns provider configuration, request serialization, response parsing, and provider selection. It must not change image metrics, classify deterministic differences, or become required for comparison.

# Connections

- [pxp orchestration](internal/pixelperfectcmd/AGENTS.md): requests advisory descriptions after deterministic analysis.
- [Image comparison](internal/imagediff/AGENTS.md): consumes region identity and bounds but remains the source of truth for metrics.

# Landmarks

- `internal/imagecontext/client.go:NewClient`: selects a configured provider.
- `internal/imagecontext/openrouter.go:OpenRouter.Describe`: requests region descriptions from OpenRouter.
- `internal/imagecontext/openai.go:OpenAI.Describe`: requests region descriptions from OpenAI.
- `internal/imagecontext/config.go:LoadProviderConfig`: resolves provider configuration.

# Boundary flows

- Information flow: `internal/imagediff/image.go:CompareImagesWithThresholds` -> `internal/imagecontext/client.go:Client` via `internal/pixelperfectcmd/command.go:NewCommand`; value: `[]imagecontext.Region`.

# Placement

Keep new providers behind the `Client` contract. Put deterministic image analysis in `imagediff` and command flags/orchestration in `pixelperfectcmd`.
