package annotations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocumentValidate(t *testing.T) {
	tests := []struct {
		name      string
		document  Document
		wantError string
	}{
		{name: "empty annotations are valid", document: Document{Version: 1, CoordinateSpace: Size{Width: 10, Height: 10}}},
		{name: "bounds may fill coordinate space", document: Document{Version: 1, CoordinateSpace: Size{Width: 10, Height: 10}, Annotations: []Annotation{{ID: "a", Bounds: Bounds{Width: 10, Height: 10}}}}},
		{name: "version is validated first", document: Document{Version: 2}, wantError: "unsupported annotations version 2"},
		{name: "width must be positive", document: Document{Version: 1, CoordinateSpace: Size{Height: 10}}, wantError: "annotation coordinate space dimensions must be positive"},
		{name: "height must be positive", document: Document{Version: 1, CoordinateSpace: Size{Width: 10}}, wantError: "annotation coordinate space dimensions must be positive"},
		{name: "id is required before bounds", document: Document{Version: 1, CoordinateSpace: Size{Width: 10, Height: 10}, Annotations: []Annotation{{}}}, wantError: "annotation id must not be empty"},
		{name: "duplicate id precedes its invalid bounds", document: Document{Version: 1, CoordinateSpace: Size{Width: 10, Height: 10}, Annotations: []Annotation{{ID: "a", Bounds: Bounds{Width: 1, Height: 1}}, {ID: "a"}}}, wantError: `duplicate annotation id "a"`},
		{name: "bounds must fit", document: Document{Version: 1, CoordinateSpace: Size{Width: 10, Height: 10}, Annotations: []Annotation{{ID: "a", Bounds: Bounds{X: 9, Width: 2, Height: 1}}}}, wantError: `annotation "a" bounds are outside coordinate space`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.document.Validate()

			if tt.wantError != "" {
				assert.EqualError(t, err, tt.wantError)
				return
			}
			assert.NoError(t, err)
		})
	}
}
