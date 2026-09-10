package annotations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAndIntersectAnnotations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "annotations.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
		"version":1,
		"coordinateSpace":{"width":100,"height":80},
		"annotations":[
			{"id":"sidebar","label":"Sidebar","bounds":{"x":0,"y":0,"width":40,"height":80},"metadata":{"source":"capture"}},
			{"id":"content","bounds":{"x":40,"y":0,"width":60,"height":80}}
		]
	}`), 0o600))

	document, err := Load(path)
	require.NoError(t, err)
	require.NoError(t, document.ValidateDimensions(100, 80))

	matches := document.Intersections(Bounds{X: 20, Y: 10, Width: 40, Height: 20})
	require.Len(t, matches, 2)
	assert.Equal(t, "sidebar", matches[0].ID)
	assert.Equal(t, 0.5, matches[0].RegionIntersectionRatio)
	assert.Equal(t, "capture", matches[0].Metadata["source"])
	assert.Equal(t, "content", matches[1].ID)
}

func TestLoadAnnotationsRejectsUnknownFieldsAndInvalidBounds(t *testing.T) {
	dir := t.TempDir()
	unknown := filepath.Join(dir, "unknown.json")
	require.NoError(t, os.WriteFile(unknown, []byte(`{"version":1,"coordinateSpace":{"width":10,"height":10},"annotations":[],"magic":true}`), 0o600))
	_, err := Load(unknown)
	assert.ErrorContains(t, err, `unknown field "magic"`)

	invalid := filepath.Join(dir, "invalid.json")
	require.NoError(t, os.WriteFile(invalid, []byte(`{"version":1,"coordinateSpace":{"width":10,"height":10},"annotations":[{"id":"bad","bounds":{"x":9,"y":0,"width":2,"height":1}}]}`), 0o600))
	_, err = Load(invalid)
	assert.ErrorContains(t, err, `annotation "bad" bounds are outside coordinate space`)
}

func TestAnnotationsRequireSupportedVersionAndMatchingDimensions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "annotations.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"version":2,"coordinateSpace":{"width":10,"height":10},"annotations":[]}`), 0o600))
	_, err := Load(path)
	assert.EqualError(t, err, "unsupported annotations version 2")

	document := Document{Version: 1, CoordinateSpace: Size{Width: 10, Height: 10}}
	assert.EqualError(t, document.ValidateDimensions(9, 10), "annotation coordinate space 10x10 does not match comparison image 9x10")
}
