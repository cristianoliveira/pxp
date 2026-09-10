package commands

import (
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnnotationsEnrichRegionsWithoutChangingMetrics(t *testing.T) {
	dir := t.TempDir()
	reference, actual := filepath.Join(dir, "reference.png"), filepath.Join(dir, "actual.png")
	referenceImage := image.NewRGBA(image.Rect(0, 0, 4, 3))
	actualImage := image.NewRGBA(image.Rect(0, 0, 4, 3))
	actualImage.Set(1, 1, color.White)
	writeTestPNG(t, reference, referenceImage)
	writeTestPNG(t, actual, actualImage)
	annotationsPath := filepath.Join(dir, "annotations.json")
	require.NoError(t, os.WriteFile(annotationsPath, []byte(`{"version":1,"coordinateSpace":{"width":4,"height":3},"annotations":[{"id":"row","label":"Selected row","bounds":{"x":0,"y":1,"width":4,"height":1}}]}`), 0o600))

	without := executeCommand(NewCommand(), reference, actual, "--output", filepath.Join(dir, "without.png"))
	with := executeCommand(NewCommand(), reference, actual, "--annotations", annotationsPath, "--output", filepath.Join(dir, "with.png"))

	require.NoError(t, without.Err)
	require.NoError(t, with.Err)
	var plain, enriched diff.ImageComparison
	require.NoError(t, json.Unmarshal([]byte(without.Stdout), &plain))
	require.NoError(t, json.Unmarshal([]byte(with.Stdout), &enriched))
	assert.Equal(t, plain.ChangedPixels, enriched.ChangedPixels)
	assert.Equal(t, plain.RMSE, enriched.RMSE)
	require.Len(t, enriched.Regions, 1)
	require.Len(t, enriched.Regions[0].Annotations, 1)
	assert.Equal(t, "row", enriched.Regions[0].Annotations[0].ID)
	assert.Equal(t, "Selected row", enriched.Regions[0].Annotations[0].Label)
}
