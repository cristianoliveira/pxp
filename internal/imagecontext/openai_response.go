package imagecontext

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type openAIResponse struct {
	Output []openAIOutput `json:"output"`
	Error  *openAIError   `json:"error"`
}

type openAIOutput struct {
	Content []openAIContent `json:"content"`
}

type openAIContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type openAIError struct {
	Message string `json:"message"`
}

// decodeOpenAIResponse leaves response body ownership with the caller.
func decodeOpenAIResponse(response *http.Response) (openAIResponse, error) {
	var envelope openAIResponse
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return openAIResponse{}, fmt.Errorf("decode visual context response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if envelope.Error != nil {
			return openAIResponse{}, fmt.Errorf("OpenAI: %s", envelope.Error.Message)
		}
		return openAIResponse{}, fmt.Errorf("OpenAI returned HTTP %d", response.StatusCode)
	}
	return envelope, nil
}

func (response openAIResponse) outputText() string {
	content := ""
	for _, output := range response.Output {
		for _, part := range output.Content {
			if part.Type == "output_text" {
				content += part.Text
			}
		}
	}
	return content
}
