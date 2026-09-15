package imagediff

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodedImagesScanBandPreservesEachHorizontalLine(t *testing.T) {
	reference := image.NewNRGBA(image.Rect(0, 0, 4, 3))
	actual := image.NewNRGBA(image.Rect(0, 0, 4, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			reference.SetNRGBA(x, y, color.NRGBA{R: uint8(y + 1), A: 255})
			actual.SetNRGBA(x, y, color.NRGBA{G: uint8(y + 1), A: 255})
		}
	}
	reference.SetNRGBA(1, 1, color.NRGBA{B: 255, A: 255})

	refLines, actualLines, err := (&DecodedImages{Reference: reference, Actual: actual}).ScanBand(ScanHorizontal, 0, 2)

	require.NoError(t, err)
	require.Len(t, refLines, 3)
	require.Len(t, actualLines, 3)
	assert.Equal(t, 0, refLines[0].Index)
	assert.Equal(t, 1, refLines[1].Index)
	assert.Equal(t, 2, refLines[2].Index)
	assert.Equal(t, 3, len(refLines[1].Runs))
	assert.Equal(t, [4]uint8{0, 0, 255, 255}, refLines[1].Runs[1].RGBA)
	assert.Equal(t, 1, len(actualLines[0].Runs))
}

func TestDecodedImagesScanBandPreservesEachVerticalLine(t *testing.T) {
	reference := image.NewNRGBA(image.Rect(0, 0, 3, 4))
	actual := image.NewNRGBA(image.Rect(0, 0, 3, 4))
	for x := 0; x < 3; x++ {
		for y := 0; y < 4; y++ {
			reference.SetNRGBA(x, y, color.NRGBA{R: uint8(x + 1), A: 255})
			actual.SetNRGBA(x, y, color.NRGBA{G: uint8(x + 1), A: 255})
		}
	}

	refLines, actualLines, err := (&DecodedImages{Reference: reference, Actual: actual}).ScanBand(ScanVertical, 1, 2)

	require.NoError(t, err)
	require.Len(t, refLines, 2)
	require.Len(t, actualLines, 2)
	assert.Equal(t, []int{1, 2}, []int{refLines[0].Index, refLines[1].Index})
	assert.Equal(t, 4, refLines[0].Runs[0].Length)
	assert.Equal(t, [4]uint8{3, 0, 0, 255}, refLines[1].Runs[0].RGBA)
}

func TestDecodedImagesScanBandRejectsReversedAndOutOfBoundsRanges(t *testing.T) {
	images := &DecodedImages{
		Reference: image.NewNRGBA(image.Rect(0, 0, 3, 3)),
		Actual:    image.NewNRGBA(image.Rect(0, 0, 3, 3)),
	}

	_, _, err := images.ScanBand(ScanHorizontal, 2, 1)
	assert.ErrorContains(t, err, "start 2 is greater than end 1")
	_, _, err = images.ScanBand(ScanVertical, 0, 3)
	var boundsErr *MeasurementBoundsError
	require.ErrorAs(t, err, &boundsErr)
	assert.Equal(t, image.Point{X: 0, Y: 0}, boundsErr.Point)
}
