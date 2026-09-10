package imagediff

import (
	"fmt"
	"image"
	"image/color"
)

// Overlay computes a transparent directional difference image in memory.
// Red pixels are stronger in the reference; green pixels are stronger in the actual image.
func (images *DecodedImages) Overlay(region *Bounds, ignored []Bounds) (*image.NRGBA, error) {
	reference, actual := images.Reference, images.Actual
	area := Bounds{Width: reference.Bounds().Dx(), Height: reference.Bounds().Dy()}
	if region != nil {
		area = *region
		if area.X < 0 || area.Y < 0 || area.Width <= 0 || area.Height <= 0 || area.X+area.Width > reference.Bounds().Dx() || area.Y+area.Height > reference.Bounds().Dy() {
			return nil, fmt.Errorf("region %d,%d,%d,%d is outside image bounds %dx%d", area.X, area.Y, area.Width, area.Height, reference.Bounds().Dx(), reference.Bounds().Dy())
		}
	}
	ignoredPixels := newIgnoredPixelMap(reference.Bounds().Dx(), reference.Bounds().Dy(), ignored)
	overlay := image.NewNRGBA(image.Rect(0, 0, area.Width, area.Height))
	for y := 0; y < area.Height; y++ {
		for x := 0; x < area.Width; x++ {
			if ignoredPixels.Contains(area.X+x, area.Y+y) {
				continue
			}
			referencePixel := color.NRGBAModel.Convert(reference.At(reference.Bounds().Min.X+area.X+x, reference.Bounds().Min.Y+area.Y+y)).(color.NRGBA)
			actualPixel := color.NRGBAModel.Convert(actual.At(actual.Bounds().Min.X+area.X+x, actual.Bounds().Min.Y+area.Y+y)).(color.NRGBA)
			referenceStrength := directionalDifference(referencePixel, actualPixel)
			actualStrength := directionalDifference(actualPixel, referencePixel)
			overlay.SetNRGBA(x, y, color.NRGBA{R: referenceStrength, G: actualStrength, A: max(referenceStrength, actualStrength)})
		}
	}
	return overlay, nil
}

func directionalDifference(first, second color.NRGBA) uint8 {
	return max(positiveDifference(first.R, second.R), positiveDifference(first.G, second.G), positiveDifference(first.B, second.B), positiveDifference(first.A, second.A))
}

func positiveDifference(first, second uint8) uint8 {
	if first <= second {
		return 0
	}
	return first - second
}
