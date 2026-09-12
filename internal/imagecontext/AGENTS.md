# Purpose

`internal/imagecontext` provides optional visual descriptions for comparison regions through OpenRouter or OpenAI-compatible providers.

# Boundaries

It owns provider configuration, request/response serialization, selection, and the advisory client contract. It must not change deterministic metrics, classify differences, or become required for a comparison.

# Connections

- [Command orchestration](internal/commands/AGENTS.md): enables and configures advisory descriptions after comparison.
- [Image comparison](internal/imagediff/AGENTS.md): supplies region identity and bounds that providers describe.

# Landmarks

- `internal/imagecontext/client.go:NewClient`: selects a configured provider client.
- `internal/imagecontext/openrouter.go:OpenRouter.Describe`: sends region context to OpenRouter.
- `internal/imagecontext/openai.go:OpenAI.Describe`: sends region context to OpenAI.
- `internal/imagecontext/config.go:LoadProviderConfig`: resolves provider configuration.

# Boundary flows

- Information flow: `internal/imageio/imageio.go:CompareImagesWithThresholds` -> `internal/imagecontext/openrouter.go:OpenRouter.Describe` via `internal/commands/command.go:NewCommand`; value: `[]imagecontext.Region`.

# Placement

Keep each provider behind the shared client contract. Put deterministic analysis in `imagediff` and command flags in `commands`.