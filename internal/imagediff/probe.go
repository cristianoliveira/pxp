package imagediff

import (
	"fmt"
	"image"
)

// MeasurementBoundsError reports a point outside a decoded image pair.
type MeasurementBoundsError struct {
	Point image.Point
	Size  image.Point
}

func (err *MeasurementBoundsError) Error() string {
	return fmt.Sprintf("point %d,%d is outside image bounds %dx%d", err.Point.X, err.Point.Y, err.Size.X, err.Size.Y)
}

// PixelProbe contains the normalized RGBA values at one point in both images.
type PixelProbe struct {
	Reference [4]uint8
	Actual    [4]uint8
}

// Probe measures one point in both decoded images.
func (images *DecodedImages) Probe(point image.Point) (PixelProbe, error) {
	if err := validateMeasurementImages(images); err != nil {
		return PixelProbe{}, err
	}
	bounds := images.Reference.Bounds()
	if !point.In(bounds) {
		return PixelProbe{}, &MeasurementBoundsError{Point: point, Size: bounds.Size()}
	}
	return PixelProbe{
		Reference: rgbaAt(images.Reference, point),
		Actual:    rgbaAt(images.Actual, point),
	}, nil
}

func rgbaAt(img image.Image, point image.Point) [4]uint8 {
	r, g, b, a := img.At(point.X, point.Y).RGBA()
	return [4]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

func validateMeasurementImages(images *DecodedImages) error {
	if images == nil || images.Reference == nil || images.Actual == nil {
		return fmt.Errorf("decoded image pair is incomplete")
	}
	if images.Reference.Bounds().Size() != images.Actual.Bounds().Size() {
		reference := images.Reference.Bounds().Size()
		actual := images.Actual.Bounds().Size()
		return fmt.Errorf("image dimensions differ: reference is %dx%d, actual is %dx%d", reference.X, reference.Y, actual.X, actual.Y)
	}
	return nil
}
