package imagediff

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodedImagesProbeReportsColorsAndAlpha(t *testing.T) {
	images := &DecodedImages{
		Reference: image.NewNRGBA(image.Rect(0, 0, 2, 1)),
		Actual:    image.NewNRGBA(image.Rect(0, 0, 2, 1)),
	}
	images.Reference.SetNRGBA(0, 0, color.NRGBA{R: 200, G: 100, B: 50, A: 128})
	images.Actual.SetNRGBA(0, 0, color.NRGBA{R: 100, G: 80, B: 40, A: 64})

	result, err := images.Probe(image.Point{X: 0, Y: 0})

	require.NoError(t, err)
	assert.Equal(t, [4]uint8{100, 50, 25, 128}, result.Reference)
	assert.Equal(t, [4]uint8{25, 20, 10, 64}, result.Actual)
}

func TestDecodedImagesProbeReportsIdenticalPixels(t *testing.T) {
	images := &DecodedImages{
		Reference: image.NewNRGBA(image.Rect(0, 0, 1, 1)),
		Actual:    image.NewNRGBA(image.Rect(0, 0, 1, 1)),
	}
	pixel := color.NRGBA{R: 12, G: 34, B: 56, A: 255}
	images.Reference.SetNRGBA(0, 0, pixel)
	images.Actual.SetNRGBA(0, 0, pixel)

	result, err := images.Probe(image.Point{X: 0, Y: 0})

	require.NoError(t, err)
	assert.Equal(t, result.Reference, result.Actual)
}

func TestDecodedImagesProbeRejectsOutOfBoundsPoint(t *testing.T) {
	images := &DecodedImages{
		Reference: image.NewNRGBA(image.Rect(0, 0, 2, 1)),
		Actual:    image.NewNRGBA(image.Rect(0, 0, 2, 1)),
	}

	_, err := images.Probe(image.Point{X: 2, Y: 0})

	var boundsErr *MeasurementBoundsError
	require.ErrorAs(t, err, &boundsErr)
	assert.Equal(t, image.Point{X: 2, Y: 0}, boundsErr.Point)
	assert.Equal(t, image.Point{X: 2, Y: 1}, boundsErr.Size)
}
