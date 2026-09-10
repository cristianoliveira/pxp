package annotations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntersections(t *testing.T) {
	document := Document{Version: 1, CoordinateSpace: Size{Width: 100, Height: 80}, Annotations: []Annotation{
		{ID: "sidebar", Label: "Sidebar", Bounds: Bounds{X: 0, Y: 0, Width: 40, Height: 80}, Metadata: map[string]any{"source": "capture"}},
		{ID: "content", Bounds: Bounds{X: 40, Y: 0, Width: 60, Height: 80}},
	}}
	require.NoError(t, document.Validate())
	matches := document.Intersections(Bounds{X: 20, Y: 10, Width: 40, Height: 20})
	require.Len(t, matches, 2)
	assert.Equal(t, "sidebar", matches[0].ID)
	assert.Equal(t, 0.5, matches[0].RegionIntersectionRatio)
	assert.Equal(t, "capture", matches[0].Metadata["source"])
	assert.Equal(t, "content", matches[1].ID)
}
