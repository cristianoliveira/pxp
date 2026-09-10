package smoke

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPixelPerfectHandlesThousandsOfDisconnectedChanges(t *testing.T) {
	binary := buildCommand(t, "pxp")
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	base := image.NewRGBA(image.Rect(0, 0, 512, 512))
	changed := image.NewRGBA(base.Bounds())
	changedPixels := 0
	for y := 1; y < 512; y += 4 {
		for x := 1; x < 512; x += 4 {
			changed.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
			changedPixels++
		}
	}
	writeSmokePNG(t, reference, base)
	writeSmokePNG(t, actual, changed)

	output, err := pixelPerfectCommand(binary, reference, actual, "--output", mask).CombinedOutput()

	require.NoError(t, err, string(output))
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal(output, &comparison))
	assert.Equal(t, 512*512, comparison.ComparedPixels)
	assert.Equal(t, changedPixels, comparison.ChangedPixels)
	require.Len(t, comparison.Regions, 20)
	for _, region := range comparison.Regions {
		assert.Equal(t, 1, region.ChangedPixels)
		assert.Equal(t, 1, region.Bounds.Width)
		assert.Equal(t, 1, region.Bounds.Height)
	}
	assertPNGDimensions(t, mask, 512, 512)
}

func TestPixelPerfectReportsMixedForCompetingMismatchSignals(t *testing.T) {
	binary := buildCommand(t, "pxp")
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	base := image.NewRGBA(image.Rect(0, 0, 256, 256))
	changed := image.NewRGBA(base.Bounds())
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			mismatch := color.RGBA{R: 180, A: 255}
			if x >= 128 {
				mismatch = color.RGBA{G: 180, A: 255}
			}
			changed.SetRGBA(x, y, mismatch)
		}
	}
	writeSmokePNG(t, reference, base)
	writeSmokePNG(t, actual, changed)

	output, err := pixelPerfectCommand(binary, reference, actual, "--output", mask).CombinedOutput()

	require.NoError(t, err, string(output))
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal(output, &comparison))
	require.Len(t, comparison.Regions, 1)
	region := comparison.Regions[0]
	assert.Equal(t, "mixed", region.Classification)
	assert.Equal(t, 256*256, region.ChangedPixels)
	require.Len(t, region.DominantColorPairs, 2)
	assert.Equal(t, region.DominantColorPairs[0].Pixels, region.DominantColorPairs[1].Pixels)
	assert.Less(t, region.EdgeRMSE, region.RMSE*0.5)
}

func TestPixelPerfectGroupingCanTurnClearSignalsIntoMixedDiagnosis(t *testing.T) {
	binary := buildCommand(t, "pxp")
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	base := image.NewRGBA(image.Rect(0, 0, 205, 100))
	changed := image.NewRGBA(base.Bounds())
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			changed.SetRGBA(x, y, color.RGBA{R: 160, A: 255})
		}
		for x := 105; x < 205; x++ {
			changed.SetRGBA(x, y, color.RGBA{G: 160, A: 255})
		}
	}
	writeSmokePNG(t, reference, base)
	writeSmokePNG(t, actual, changed)

	compare := func(t *testing.T, flags ...string) diff.ImageComparison {
		t.Helper()
		args := []string{reference, actual, "--output", filepath.Join(t.TempDir(), "mask.png")}
		args = append(args, flags...)
		output, err := pixelPerfectCommand(binary, args...).CombinedOutput()
		require.NoError(t, err, string(output))
		var comparison diff.ImageComparison
		require.NoError(t, json.Unmarshal(output, &comparison))
		return comparison
	}

	separate := compare(t)
	require.Len(t, separate.Regions, 2)
	assert.Equal(t, "solid-fill", separate.Regions[0].Classification)
	assert.Equal(t, "solid-fill", separate.Regions[1].Classification)

	grouped := compare(t, "--region-gap", "5")
	require.Len(t, grouped.Regions, 1)
	assert.Equal(t, "mixed", grouped.Regions[0].Classification)
	assert.Equal(t, 20_000, grouped.Regions[0].ChangedPixels)
	require.Len(t, grouped.Regions[0].DominantColorPairs, 2)
}

func TestPixelPerfectExposesClassificationBoundarySensitivity(t *testing.T) {
	binary := buildCommand(t, "pxp")
	for _, test := range []struct {
		name           string
		dominantPixels int
		expected       string
	}{
		{name: "below dominant color boundary", dominantPixels: 699, expected: "mixed"},
		{name: "at dominant color boundary", dominantPixels: 700, expected: "solid-fill"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			reference := filepath.Join(dir, "reference.png")
			actual := filepath.Join(dir, "actual.png")
			mask := filepath.Join(dir, "mask.png")
			base := image.NewRGBA(image.Rect(0, 0, 1000, 1))
			changed := image.NewRGBA(base.Bounds())
			for x := 0; x < 1000; x++ {
				mismatch := color.RGBA{G: 120, A: 255}
				if x < test.dominantPixels {
					mismatch = color.RGBA{R: 120, A: 255}
				}
				changed.SetRGBA(x, 0, mismatch)
			}
			writeSmokePNG(t, reference, base)
			writeSmokePNG(t, actual, changed)

			output, err := pixelPerfectCommand(binary, reference, actual, "--output", mask).CombinedOutput()
			require.NoError(t, err, string(output))
			var comparison diff.ImageComparison
			require.NoError(t, json.Unmarshal(output, &comparison))
			require.Len(t, comparison.Regions, 1)
			assert.Equal(t, test.expected, comparison.Regions[0].Classification)
			assert.Equal(t, test.dominantPixels, comparison.Regions[0].DominantColorPairs[0].Pixels)
		})
	}
}

func TestPixelPerfectHandlesDenseAlphaGradient(t *testing.T) {
	binary := buildCommand(t, "pxp")
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	overlay := filepath.Join(dir, "overlay.png")
	base := image.NewRGBA(image.Rect(0, 0, 1024, 512))
	changed := image.NewRGBA(base.Bounds())
	for y := 0; y < 512; y++ {
		for x := 0; x < 1024; x++ {
			alpha := uint8(x % 256)
			base.SetRGBA(x, y, color.RGBA{R: 40, G: 80, B: 120, A: alpha})
			changed.SetRGBA(x, y, color.RGBA{R: 42, G: 78, B: 125, A: alpha})
		}
	}
	writeSmokePNG(t, reference, base)
	writeSmokePNG(t, actual, changed)

	output, err := pixelPerfectCommand(binary, reference, actual, "--output", mask, "--overlay", overlay, "--threshold", "1").CombinedOutput()

	require.NoError(t, err, string(output))
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal(output, &comparison))
	assert.Equal(t, 1024*512, comparison.ComparedPixels)
	assert.Greater(t, comparison.ChangedPixels, 500_000)
	assert.Greater(t, comparison.RGBRMSE, 0.0)
	assert.Equal(t, 0.0, comparison.AlphaRMSE)
	assertPNGDimensions(t, mask, 1024, 512)
	assertPNGDimensions(t, overlay, 1024, 512)
}

func writeSmokePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(file, img))
	require.NoError(t, file.Close())
}
