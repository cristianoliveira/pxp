package imagediff

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareReturnsMaskPixelsWithoutWritingFiles(t *testing.T) {
	reference := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	actual := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	actual.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})

	result, mask, err := (&DecodedImages{Reference: reference, Actual: actual}).Compare(0, DefaultPerceptualThreshold, nil, nil)

	require.NoError(t, err)
	require.Equal(t, 1, result.ChangedPixels)
	require.NotNil(t, mask)
	require.Equal(t, uint8(255), mask.NRGBAAt(0, 0).R)
}
