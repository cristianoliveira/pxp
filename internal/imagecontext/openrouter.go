package imagecontext

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const DefaultModel = "google/gemini-2.5-flash"
const DefaultBaseURL = "https://openrouter.ai/api/v1"

type Bounds struct{ X, Y, Width, Height int }
type Region struct {
	ID     string
	Bounds Bounds
}
type Input struct {
	ReferencePath, ActualPath string
	Regions                   []Region
	Prompt                    string
}
type RegionContext struct {
	Region              string `json:"region"`
	ReferenceAppearance string `json:"referenceAppearance,omitempty"`
	ActualAppearance    string `json:"actualAppearance,omitempty"`
	VisualContext       string `json:"visualContext,omitempty"`
}
type Result struct {
	Provider   string          `json:"provider"`
	Model      string          `json:"model,omitempty"`
	Advisory   bool            `json:"advisory"`
	Disclaimer string          `json:"disclaimer,omitempty"`
	Prompt     string          `json:"prompt,omitempty"`
	Regions    []RegionContext `json:"regions,omitempty"`
}
type OpenRouter struct {
	apiKey, model, baseURL string
	client                 *http.Client
}

func NewOpenRouter(apiKey, model, baseURL string) *OpenRouter {
	return &OpenRouter{
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  http.DefaultClient,
	}
}

func (o *OpenRouter) Describe(ctx context.Context, input Input) (Result, error) {
	ref, err := imageDataURL(input.ReferencePath)
	if err != nil {
		return Result{}, fmt.Errorf("read reference for visual context: %w", err)
	}
	actual, err := imageDataURL(input.ActualPath)
	if err != nil {
		return Result{}, fmt.Errorf("read actual for visual context: %w", err)
	}
	prompt := visualContextPrompt(input.Regions, input.Prompt)
	payload := map[string]any{
		"model":  o.model,
		"stream": false,
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": prompt},
					map[string]any{"type": "image_url", "image_url": map[string]any{"url": ref}},
					map[string]any{"type": "image_url", "image_url": map[string]any{"url": actual}},
				},
			},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		o.baseURL+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Title", "pxp")
	response, err := o.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("request visual context: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return Result{}, fmt.Errorf("decode visual context response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if envelope.Error != nil {
			return Result{}, fmt.Errorf("OpenRouter: %s", envelope.Error.Message)
		}
		return Result{}, fmt.Errorf("OpenRouter returned HTTP %d", response.StatusCode)
	}
	if len(envelope.Choices) == 0 {
		return Result{}, fmt.Errorf("OpenRouter response did not include visual context")
	}
	regions, err := parseRegionContexts(envelope.Choices[0].Message.Content, input.Regions)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Provider: "openrouter",
		Model:    o.model,
		Advisory: true,
		Prompt:   input.Prompt,
		Regions:  regions,
	}, nil
}
func visualContextPrompt(regions []Region, customPrompt string) string {
	encoded, _ := json.Marshal(regions)
	prompt := `The first image is reference and second is implementation. For each ` +
		`supplied region ID, name the visible object and briefly describe its appearance ` +
		`in each image. Mention only differences clearly visible inside that region. Use ` +
		`at most 15 words per field. Do not describe causes, measure, diagnose geometry, ` +
		`suggest fixes, infer DOM/domain semantics, or alter metrics. Do not mention ` +
		`anything outside the supplied region. Preserve region IDs exactly. Return JSON ` +
		`only: {"regions":[{"region":"r1","referenceAppearance":"","actualAppearance":"",` +
		`"visualContext":""}]}.`
	if strings.TrimSpace(customPrompt) != "" {
		prompt += " User focus: " + strings.TrimSpace(customPrompt)
	}
	return prompt + " Regions: " + string(encoded)
}

func parseRegionContexts(content string, expected []Region) ([]RegionContext, error) {
	content = strings.TrimSpace(
		strings.TrimSuffix(
			strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(content), "```json"), "```"),
			"```",
		),
	)
	var parsed struct {
		Regions []RegionContext `json:"regions"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("decode visual context JSON: %w", err)
	}
	allowed := map[string]bool{}
	for _, region := range expected {
		allowed[region.ID] = true
	}
	for _, region := range parsed.Regions {
		if !allowed[region.Region] {
			return nil, fmt.Errorf("visual context returned unknown region %q", region.Region)
		}
	}
	return parsed.Regions, nil
}

func imageDataURL(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data), nil
}
