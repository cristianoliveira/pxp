package imagecontext

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeOpenAIResponse(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantText  string
		wantError string
	}{
		{name: "joins output text in order and ignores other content", status: 200, body: `{"output":[{"content":[{"type":"reasoning","text":"ignore"},{"type":"output_text","text":"first"},{"type":"output_text","text":" second"}]},{"content":[{"type":"output_text","text":" third"}]}]}`, wantText: "first second third"},
		{name: "empty successful response", status: 299, body: `{}`},
		{name: "provider error", status: 400, body: `{"error":{"message":"invalid request"}}`, wantError: "OpenAI: invalid request"},
		{name: "status without provider error", status: 500, body: `{}`, wantError: "OpenAI returned HTTP 500"},
		{name: "redirect is not success", status: 300, body: `{}`, wantError: "OpenAI returned HTTP 300"},
		{name: "informational status is not success", status: 199, body: `{}`, wantError: "OpenAI returned HTTP 199"},
		{name: "malformed success body", status: 200, body: `{`, wantError: "decode visual context response: unexpected EOF"},
		{name: "decode error precedes status error", status: 502, body: `{`, wantError: "decode visual context response: unexpected EOF"},
		{name: "empty provider message remains provider error", status: 400, body: `{"error":{}}`, wantError: "OpenAI: "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &http.Response{StatusCode: tt.status, Body: io.NopCloser(strings.NewReader(tt.body))}

			envelope, err := decodeOpenAIResponse(response)

			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantText, envelope.outputText())
		})
	}
}
