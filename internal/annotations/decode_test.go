package annotations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeAnnotationsFromReader(t *testing.T) {
	reader := strings.NewReader(`{"version":1,"coordinateSpace":{"width":10,"height":20},"annotations":[{"id":"a","bounds":{"x":0,"y":0,"width":10,"height":20}}]}`)

	document, err := Decode(reader)

	require.NoError(t, err)
	assert.Equal(t, &Document{Version: 1, CoordinateSpace: Size{Width: 10, Height: 20}, Annotations: []Annotation{{ID: "a", Bounds: Bounds{Width: 10, Height: 20}}}}, document)
}

func TestDecodeAnnotationsRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantError string
	}{
		{name: "malformed JSON", body: `{`, wantError: "decode annotations: unexpected EOF"},
		{name: "unknown field", body: `{"magic":true}`, wantError: `decode annotations: json: unknown field "magic"`},
		{name: "multiple values precede validation", body: `{} {}`, wantError: "decode annotations: multiple JSON values"},
		{name: "invalid trailing JSON precedes validation", body: `{} {`, wantError: "decode annotations: unexpected EOF"},
		{name: "domain validation", body: `{"version":2}`, wantError: "unsupported annotations version 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			document, err := Decode(strings.NewReader(tt.body))

			require.EqualError(t, err, tt.wantError)
			assert.Nil(t, document)
		})
	}
}
