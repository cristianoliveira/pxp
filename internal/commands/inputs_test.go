package commands

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	clipkg "github.com/cristianoliveira/pxp/internal/cli"
	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareComparisonInputsRejectsConflictingCropBeforeFileAccess(t *testing.T) {
	command := newCommand(nil)
	require.NoError(t, command.Flags().Set("reference-crop", "0,0,1,1"))
	require.NoError(t, command.Flags().Set("reference-metadata", "missing.json"))

	_, _, err := prepareComparisonInputs(command, []string{"missing-reference.png", "missing-actual.png"}, nil)

	require.Error(t, err)
	assert.Equal(t, 2, clipkg.ExitCode(err))
	assert.ErrorContains(t, err, "--reference-crop and --reference-metadata cannot be used together")
	assert.NotContains(t, err.Error(), "missing.json")
}

func TestLoadExportMetadataAcceptsFractionalLogicalCrop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reference.export.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"version":1,"nodeBounds":{"width":780,"height":72},"exportBounds":{"width":780,"height":74},"logicalCrop":{"x":26.9871,"y":16,"width":780,"height":72}}`), 0o600))

	metadata, err := loadExportMetadata(path)

	require.NoError(t, err)
	assert.Equal(t, &diff.Bounds{X: 0, Y: 1, Width: 780, Height: 72}, cropFromExportMetadata(*metadata))
}

func TestDiffImageCommandAppliesReferenceMetadataCrop(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	metadata := filepath.Join(dir, "reference.export.json")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 4, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	require.NoError(t, os.WriteFile(metadata, []byte(`{"version":1,"nodeBounds":{"width":2,"height":2},"exportBounds":{"width":4,"height":2},"logicalCrop":{"x":1,"y":0,"width":2,"height":2}}`), 0o600))

	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--reference-metadata", metadata, "--output", mask)

	require.NoError(t, result.Err)
	assert.JSONEq(t, `{"width":2,"height":2,"changedPixels":0,"comparedPixels":4,"changedRatio":0,"rmse":0,"rgbRmse":0,"luminanceRmse":0,"alphaRmse":0,"edgeRmse":0,"perceptualRmse":0,"perceptualChangedPixels":0,"perceptualChangedRatio":0,"perceptualThreshold":0.1,"antialiasedPixels":0,"evidence":{"rawOnlyPixels":0,"perceptualOnlyPixels":0,"rawAndPerceptualPixels":0},"inputs":{"reference":{"width":4,"height":2,"crop":{"x":1,"y":0,"width":2,"height":2}},"actual":{"width":2,"height":2}},"mask":"`+mask+`"}`, result.Stdout)
}

func TestDiffImageCommandRejectsUnsupportedReferenceMetadataVersion(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	metadata := filepath.Join(dir, "reference.export.json")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 4, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	require.NoError(t, os.WriteFile(metadata, []byte(`{"version":2}`), 0o600))

	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--reference-metadata", metadata, "--output", filepath.Join(dir, "mask.png"))

	assert.EqualError(t, result.Err, "unsupported --reference-metadata version 2")
}

func TestDiffImageCommandRejectsReferenceMetadataDimensionMismatch(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	metadata := filepath.Join(dir, "reference.export.json")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 4, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	require.NoError(t, os.WriteFile(metadata, []byte(`{"version":1,"nodeBounds":{"width":2,"height":2},"exportBounds":{"width":5,"height":2}}`), 0o600))

	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--reference-metadata", metadata, "--output", filepath.Join(dir, "mask.png"))

	assert.EqualError(t, result.Err, "--reference-metadata export bounds 5x2 do not match reference image 4x2")
}

func TestDiffImageCommandAppliesIndependentInputCrops(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 4, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 2, 2)))

	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--reference-crop", "1,0,2,2", "--actual-crop", "0,0,2,2", "--output", mask)

	require.NoError(t, result.Err)
	assert.JSONEq(t, `{"width":2,"height":2,"changedPixels":0,"comparedPixels":4,"changedRatio":0,"rmse":0,"rgbRmse":0,"luminanceRmse":0,"alphaRmse":0,"edgeRmse":0,"perceptualRmse":0,"perceptualChangedPixels":0,"perceptualChangedRatio":0,"perceptualThreshold":0.1,"antialiasedPixels":0,"evidence":{"rawOnlyPixels":0,"perceptualOnlyPixels":0,"rawAndPerceptualPixels":0},"inputs":{"reference":{"width":4,"height":2,"crop":{"x":1,"y":0,"width":2,"height":2}},"actual":{"width":2,"height":2,"crop":{"x":0,"y":0,"width":2,"height":2}}},"mask":"`+mask+`"}`, result.Stdout)
}

func TestDiffImageCommandAddsInputBoundsToRegionsWhenCropped(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	referenceImage := image.NewRGBA(image.Rect(0, 0, 4, 2))
	actualImage := image.NewRGBA(image.Rect(0, 0, 2, 2))
	actualImage.Set(1, 1, image.White)
	writeTestPNG(t, reference, referenceImage)
	writeTestPNG(t, actual, actualImage)

	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--reference-crop", "1,0,2,2", "--actual-crop", "0,0,2,2", "--output", mask)

	require.NoError(t, result.Err)
	region := extractFirstRegion(t, result.Stdout)
	assert.Equal(t, diff.Bounds{X: 1, Y: 1, Width: 1, Height: 1}, region.Bounds)
	assert.Equal(t, &diff.InputBounds{
		Reference: diff.Bounds{X: 2, Y: 1, Width: 1, Height: 1},
		Actual:    diff.Bounds{X: 1, Y: 1, Width: 1, Height: 1},
	}, region.InputBounds)
}

func TestDiffImageCommandRejectsInvalidCrops(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 4, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 2, 2)))

	tests := []struct {
		name, flag, crop, expected string
	}{
		{name: "negative", flag: "--reference-crop", crop: "-1,0,2,2", expected: "invalid --reference-crop: crop -1,0,2,2 is outside image bounds 4x2"},
		{name: "zero", flag: "--reference-crop", crop: "0,0,0,2", expected: "invalid --reference-crop: width and height must be positive"},
		{name: "out of bounds", flag: "--actual-crop", crop: "1,0,2,2", expected: "invalid --actual-crop: crop 1,0,2,2 is outside image bounds 2x2"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, test.flag, test.crop, "--output", filepath.Join(dir, test.name+".png"))

			assert.EqualError(t, result.Err, test.expected)
		})
	}
}

func TestDiffImageCommandRejectsCroppedDimensionMismatch(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 4, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 4, 2)))

	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--reference-crop", "0,0,2,2", "--actual-crop", "0,0,3,2", "--output", filepath.Join(dir, "mask.png"))

	assert.EqualError(t, result.Err, "cropped image dimensions differ: reference is 2x2, actual is 3x2")
}
