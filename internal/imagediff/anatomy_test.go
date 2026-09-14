package imagediff

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverAnatomyFindsStableReadingOrderAndMetrics(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 10, 6))
	fillNRGBA(img, color.White)
	black := color.NRGBA{A: 255}
	img.SetNRGBA(1, 1, black)
	img.SetNRGBA(2, 1, black)
	img.SetNRGBA(1, 2, black)
	img.SetNRGBA(2, 2, black)
	for x := 6; x < 8; x++ {
		img.SetNRGBA(x, 4, color.NRGBAModel.Convert(color.RGBA{R: 15, G: 108, B: 189, A: 255}).(color.NRGBA))
	}

	result, err := DiscoverAnatomy(img, AnatomyOptions{Threshold: 8, MinPixels: 1})

	require.NoError(t, err)
	assert.Equal(t, "#FFFFFF", result.Background)
	require.Len(t, result.Elements, 2)
	assert.Equal(t, Bounds{X: 1, Y: 1, Width: 2, Height: 2}, result.Elements[0].Bounds)
	assert.Equal(t, 4, result.Elements[0].Pixels)
	assert.Equal(t, 1.0, result.Elements[0].Density)
	assert.Equal(t, "#000000", result.Elements[0].DominantColor)
	assert.Equal(t, Bounds{X: 6, Y: 4, Width: 2, Height: 1}, result.Elements[1].Bounds)
	assert.Equal(t, "#0F6CBD", result.Elements[1].DominantColor)
}

func TestDiscoverAnatomyFiltersAndGroupsDeterministically(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 12, 4))
	fillNRGBA(img, color.White)
	black := color.NRGBA{A: 255}
	for x := 1; x < 3; x++ {
		img.SetNRGBA(x, 1, black)
		img.SetNRGBA(x+4, 1, black)
	}

	first, err := DiscoverAnatomy(img, AnatomyOptions{MinPixels: 2, Group: 3})
	require.NoError(t, err)
	second, err := DiscoverAnatomy(img, AnatomyOptions{MinPixels: 2, Group: 3})
	require.NoError(t, err)
	assert.Equal(t, first, second)
	require.Len(t, first.Elements, 1)
	assert.Equal(t, Bounds{X: 1, Y: 1, Width: 6, Height: 1}, first.Elements[0].Bounds)
	assert.Equal(t, 4, first.Elements[0].Pixels)
}

func TestDiscoverAnatomySolidImageReturnsExplicitEmptyResult(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	fillNRGBA(img, color.White)

	result, err := DiscoverAnatomy(img, AnatomyOptions{MinPixels: 1})

	require.NoError(t, err)
	assert.Equal(t, AnatomyQuery, result.Query)
	assert.Equal(t, 0, result.Total)
	assert.Equal(t, 0, result.Returned)
	assert.Contains(t, result.Message, "no foreground-elements found")
}

func TestDiscoverAnatomyHonoursExplicitBackground(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	fillNRGBA(img, color.Black)
	img.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	background := [4]uint8{255, 255, 255, 255}

	result, err := DiscoverAnatomy(img, AnatomyOptions{Background: &background, MinPixels: 1})

	require.NoError(t, err)
	require.Len(t, result.Elements, 1)
	assert.Equal(t, Bounds{X: 0, Y: 0, Width: 3, Height: 3}, result.Elements[0].Bounds)
}

func fillNRGBA(img *image.NRGBA, value color.Color) {
	pixel := color.NRGBAModel.Convert(value).(color.NRGBA)
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			img.SetNRGBA(x, y, pixel)
		}
	}
}
