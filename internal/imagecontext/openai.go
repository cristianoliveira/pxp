package imagecontext

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const DefaultOpenAIModel = "gpt-5.4-mini"
const DefaultOpenAIBaseURL = "https://api.openai.com/v1"

type OpenAI struct {
	apiKey, model, baseURL string
	client                 *http.Client
}

func NewOpenAI(apiKey, model, baseURL string) *OpenAI {
	return &OpenAI{apiKey: apiKey, model: model, baseURL: strings.TrimRight(baseURL, "/"), client: http.DefaultClient}
}
func (o *OpenAI) Describe(ctx context.Context, input Input) (Result, error) {
	ref, err := imageDataURL(input.ReferencePath)
	if err != nil {
		return Result{}, fmt.Errorf("read reference for visual context: %w", err)
	}
	actual, err := imageDataURL(input.ActualPath)
	if err != nil {
		return Result{}, fmt.Errorf("read actual for visual context: %w", err)
	}
	prompt := visualContextPrompt(input.Regions, input.Prompt)
	payload := map[string]any{"model": o.model, "input": []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "input_text", "text": prompt}, map[string]any{"type": "input_image", "image_url": ref}, map[string]any{"type": "input_image", "image_url": actual}}}}}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")
	response, err := o.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("request visual context: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	envelope, err := decodeOpenAIResponse(response)
	if err != nil {
		return Result{}, err
	}
	regions, err := parseRegionContexts(envelope.outputText(), input.Regions)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Provider: "openai",
		Model:    o.model,
		Advisory: true,
		Prompt:   input.Prompt,
		Regions:  regions,
	}, nil
}
