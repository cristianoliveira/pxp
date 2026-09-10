package annotations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocumentValidateKeepsDomainRulesIndependentOfPersistence(t *testing.T) {
	tests := []struct {
		name      string
		document  Document
		wantError string
	}{
		{name: "valid document", document: Document{Version: 1, CoordinateSpace: Size{Width: 10, Height: 10}}},
		{name: "unsupported version", document: Document{Version: 2}, wantError: "unsupported annotations version 2"},
		{name: "invalid coordinate space", document: Document{Version: 1, CoordinateSpace: Size{Width: 0, Height: 10}}, wantError: "annotation coordinate space dimensions must be positive"},
		{name: "duplicate identifier", document: Document{Version: 1, CoordinateSpace: Size{Width: 10, Height: 10}, Annotations: []Annotation{{ID: "a", Bounds: Bounds{Width: 1, Height: 1}}, {ID: "a"}}}, wantError: `duplicate annotation id "a"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.document.Validate()
			if tt.wantError == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.wantError)
		})
	}
}
