package smoke

import (
	"encoding/json"
	"path/filepath"
	"testing"

	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPixelPerfectAgainstUpstreamComparisonCorpus(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "upstream")
	tests := []struct {
		name, project, reference, actual string
		width, height                    int
		changed, regions                 int
		assertMetrics                    func(*testing.T, diff.ImageComparison)
	}{
		{
			name: "pixelmatch realistic antialias and transparency", project: "pixelmatch", reference: "1a.png", actual: "1b.png",
			width: 512, height: 256, changed: 12_933, regions: 14,
			assertMetrics: func(t *testing.T, result diff.ImageComparison) {
				assert.Greater(t, result.RGBRMSE, 0.0)
				assert.Greater(t, result.EdgeRMSE, 0.0)
				assert.Greater(t, result.PerceptualRMSE, 0.0)
				assert.Equal(t, 178, result.PerceptualChangedPixels)
				assert.Less(t, result.PerceptualChangedPixels, result.ChangedPixels)
				assert.Equal(t, 2670, result.AntialiasedPixels)
				regionalAntialias := 0
				for _, region := range result.Regions {
					regionalAntialias += region.AntialiasedPixels
				}
				assert.Equal(t, result.AntialiasedPixels, regionalAntialias)
			},
		},
		{
			name: "looks-same antialias-only raster variation", project: "looks-same", reference: "antialiasing-ref.png", actual: "antialiasing-actual.png",
			width: 12, height: 15, changed: 5, regions: 5,
			assertMetrics: func(t *testing.T, result diff.ImageComparison) {
				assert.Greater(t, result.EdgeRMSE, result.RGBRMSE)
				assert.Greater(t, result.PerceptualRMSE, 0.0)
				assert.Equal(t, 2, result.PerceptualChangedPixels)
				assert.Less(t, result.PerceptualChangedPixels, result.ChangedPixels)
				assert.Equal(t, result.ChangedPixels, result.AntialiasedPixels)
				for _, region := range result.Regions {
					assert.Equal(t, region.ChangedPixels, region.AntialiasedPixels)
				}
			},
		},
		{
			name: "odiff equivalent extreme alpha encodings", project: "odiff", reference: "extreme-alpha.png", actual: "extreme-alpha-1.png",
			width: 450, height: 450,
			assertMetrics: func(t *testing.T, result diff.ImageComparison) {
				assert.Zero(t, result.RMSE)
				assert.Zero(t, result.AlphaRMSE)
				assert.Zero(t, result.PerceptualRMSE)
				assert.Zero(t, result.PerceptualChangedPixels)
			},
		},
		{
			name: "odiff realistic RGB and RGBA map", project: "odiff", reference: "test-map-reference.png", actual: "test-map-actual.png",
			width: 438, height: 412, changed: 69_117, regions: 1,
			assertMetrics: func(t *testing.T, result diff.ImageComparison) {
				assert.Zero(t, result.AlphaRMSE)
				assert.Equal(t, 10_094, result.PerceptualChangedPixels)
				assert.Equal(t, 15_114, result.AntialiasedPixels)
				assert.Less(t, result.PerceptualChangedPixels, result.ChangedPixels)
				assert.Greater(t, result.EdgeRMSE, 0.0)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mask := filepath.Join(t.TempDir(), "mask.png")
			output, err := pixelPerfectCommand(binary,
				filepath.Join(fixtures, test.project, test.reference),
				filepath.Join(fixtures, test.project, test.actual),
				"--output", mask,
			).CombinedOutput()

			require.NoError(t, err, string(output))
			var comparison diff.ImageComparison
			require.NoError(t, json.Unmarshal(output, &comparison))
			assert.Equal(t, test.width, comparison.Width)
			assert.Equal(t, test.height, comparison.Height)
			assert.Equal(t, test.width*test.height, comparison.ComparedPixels)
			assert.Equal(t, test.changed, comparison.ChangedPixels)
			assert.Equal(t, comparison.ChangedPixels, comparison.Evidence.RawOnlyPixels+comparison.Evidence.RawAndPerceptualPixels)
			assert.Equal(t, comparison.PerceptualChangedPixels, comparison.Evidence.PerceptualOnlyPixels+comparison.Evidence.RawAndPerceptualPixels)
			assert.Len(t, comparison.Regions, test.regions)
			assertPNGDimensions(t, mask, test.width, test.height)
			test.assertMetrics(t, comparison)
		})
	}
}
