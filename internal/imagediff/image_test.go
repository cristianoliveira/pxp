package imagediff

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareComputesChangedAreaAndMaskInMemory(t *testing.T) {
	reference := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	actual := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	actual.SetNRGBA(1, 0, color.NRGBA{R: 255, A: 255})
	actual.SetNRGBA(2, 1, color.NRGBA{B: 255, A: 255})

	result, mask, err := (&DecodedImages{Reference: reference, Actual: actual}).Compare(0, DefaultPerceptualThreshold, nil, nil)

	require.NoError(t, err)
	assert.Equal(t, 3, result.Width)
	assert.Equal(t, 2, result.Height)
	assert.Equal(t, 2, result.ChangedPixels)
	assert.InDelta(t, 1.0/3.0, result.ChangedRatio, 0.0001)
	assert.Equal(t, &Bounds{X: 1, Y: 0, Width: 2, Height: 2}, result.Bounds)
	assert.Equal(t, uint8(255), mask.NRGBAAt(1, 0).A)
	assert.Equal(t, []int{0, 1}, result.ChangedRows)
}

func TestCompareHonoursRegionAndIgnoredPixels(t *testing.T) {
	reference := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	actual := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	actual.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	actual.SetNRGBA(3, 1, color.NRGBA{R: 255, A: 255})

	result, _, err := (&DecodedImages{Reference: reference, Actual: actual}).Compare(0, DefaultPerceptualThreshold, &Bounds{X: 2, Y: 0, Width: 2, Height: 2}, nil)

	require.NoError(t, err)
	assert.Equal(t, 1, result.ChangedPixels)
	assert.Equal(t, &Bounds{X: 3, Y: 1, Width: 1, Height: 1}, result.Bounds)

	result, _, err = (&DecodedImages{Reference: reference, Actual: actual}).Compare(0, DefaultPerceptualThreshold, nil, []Bounds{{X: 3, Y: 1, Width: 1, Height: 1}})
	require.NoError(t, err)
	assert.Equal(t, 1, result.ChangedPixels)
	assert.Equal(t, &Bounds{X: 0, Y: 0, Width: 1, Height: 1}, result.Bounds)
}

func TestMeasureRegionsComputesLocalMetrics(t *testing.T) {
	reference := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	actual := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	actual.SetNRGBA(2, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	images := &DecodedImages{Reference: reference, Actual: actual}

	metrics, err := images.MeasureRegions([]Bounds{{X: 2, Y: 0, Width: 2, Height: 2}}, 0, DefaultPerceptualThreshold, nil)

	require.NoError(t, err)
	require.Len(t, metrics, 1)
	assert.Equal(t, 1, metrics[0].ChangedPixels)
	assert.InDelta(t, 0.25, metrics[0].ChangedRatio, 0.000001)
	assert.Greater(t, metrics[0].RMSE, 0.0)
}

func TestCompareRejectsInvalidRegion(t *testing.T) {
	images := &DecodedImages{Reference: image.NewNRGBA(image.Rect(0, 0, 2, 2)), Actual: image.NewNRGBA(image.Rect(0, 0, 2, 2))}
	_, _, err := images.Compare(0, DefaultPerceptualThreshold, &Bounds{X: 2, Y: 0, Width: 1, Height: 1}, nil)
	assert.EqualError(t, err, "region 2,0,1,1 is outside image bounds 2x2")
}
