package imagediff

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodedImagesScanReportsHorizontalRunsIncludingFirstAndLast(t *testing.T) {
	reference := image.NewNRGBA(image.Rect(0, 0, 5, 2))
	actual := image.NewNRGBA(image.Rect(0, 0, 5, 2))
	for x := 0; x < 5; x++ {
		reference.SetNRGBA(x, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
		actual.SetNRGBA(x, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	}
	reference.SetNRGBA(2, 0, color.NRGBA{R: 40, G: 50, B: 60, A: 255})
	actual.SetNRGBA(4, 0, color.NRGBA{R: 40, G: 50, B: 60, A: 255})

	referenceRuns, actualRuns, err := (&DecodedImages{Reference: reference, Actual: actual}).Scan(ScanHorizontal, 0)

	require.NoError(t, err)
	assert.Equal(t, []ColorRun{
		{Start: 0, End: 1, Length: 2, RGBA: [4]uint8{10, 20, 30, 255}},
		{Start: 2, End: 2, Length: 1, RGBA: [4]uint8{40, 50, 60, 255}},
		{Start: 3, End: 4, Length: 2, RGBA: [4]uint8{10, 20, 30, 255}},
	}, referenceRuns)
	assert.Equal(t, []ColorRun{
		{Start: 0, End: 3, Length: 4, RGBA: [4]uint8{10, 20, 30, 255}},
		{Start: 4, End: 4, Length: 1, RGBA: [4]uint8{40, 50, 60, 255}},
	}, actualRuns)
}

func TestDecodedImagesScanReportsVerticalRunsAndRejectsInvalidIndex(t *testing.T) {
	reference := image.NewNRGBA(image.Rect(0, 0, 2, 4))
	actual := image.NewNRGBA(image.Rect(0, 0, 2, 4))
	for y := 0; y < 4; y++ {
		reference.SetNRGBA(1, y, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
		actual.SetNRGBA(1, y, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	}
	reference.SetNRGBA(1, 3, color.NRGBA{R: 4, G: 5, B: 6, A: 255})

	referenceRuns, actualRuns, err := (&DecodedImages{Reference: reference, Actual: actual}).Scan(ScanVertical, 1)

	require.NoError(t, err)
	assert.Equal(t, []ColorRun{
		{Start: 0, End: 2, Length: 3, RGBA: [4]uint8{1, 2, 3, 255}},
		{Start: 3, End: 3, Length: 1, RGBA: [4]uint8{4, 5, 6, 255}},
	}, referenceRuns)
	assert.Equal(t, []ColorRun{{Start: 0, End: 3, Length: 4, RGBA: [4]uint8{1, 2, 3, 255}}}, actualRuns)

	_, _, err = (&DecodedImages{Reference: reference, Actual: actual}).Scan(ScanVertical, 2)

	var boundsErr *MeasurementBoundsError
	require.ErrorAs(t, err, &boundsErr)
	assert.Equal(t, image.Point{X: 2, Y: 0}, boundsErr.Point)
	assert.Equal(t, image.Point{X: 2, Y: 4}, boundsErr.Size)
}
