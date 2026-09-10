package imagediff

import (
	"fmt"
	"image"
)

// ScanAxis identifies the direction of a fixed-position scan.
type ScanAxis string

const (
	// ScanHorizontal scans left-to-right along a row.
	ScanHorizontal ScanAxis = "horizontal"
	// ScanVertical scans top-to-bottom along a column.
	ScanVertical ScanAxis = "vertical"
)

// ColorRun is one contiguous run of identical normalized RGBA pixels.
type ColorRun struct {
	Start  int
	End    int
	Length int
	RGBA   [4]uint8
}

// Scan returns color runs for the requested row or column in both images.
func (images *DecodedImages) Scan(axis ScanAxis, index int) ([]ColorRun, []ColorRun, error) {
	if err := validateMeasurementImages(images); err != nil {
		return nil, nil, err
	}
	if axis != ScanHorizontal && axis != ScanVertical {
		return nil, nil, fmt.Errorf("unsupported scan axis %q", axis)
	}
	bounds := images.Reference.Bounds()
	limit := bounds.Dy()
	length := bounds.Dx()
	if axis == ScanVertical {
		limit = bounds.Dx()
		length = bounds.Dy()
	}
	if index < 0 || index >= limit {
		point := image.Point{X: 0, Y: index}
		if axis == ScanVertical {
			point = image.Point{X: index, Y: 0}
		}
		return nil, nil, &MeasurementBoundsError{Point: point, Size: bounds.Size()}
	}
	return scanRuns(
			images.Reference,
			axis,
			index,
			length,
		), scanRuns(
			images.Actual,
			axis,
			index,
			length,
		), nil
}

func scanRuns(img image.Image, axis ScanAxis, index, length int) []ColorRun {
	bounds := img.Bounds()
	runs := make([]ColorRun, 0)
	for position := 0; position < length; position++ {
		point := image.Point{X: bounds.Min.X + position, Y: bounds.Min.Y + index}
		if axis == ScanVertical {
			point = image.Point{X: bounds.Min.X + index, Y: bounds.Min.Y + position}
		}
		pixel := rgbaAt(img, point)
		if len(runs) > 0 && runs[len(runs)-1].RGBA == pixel {
			runs[len(runs)-1].End = position
			runs[len(runs)-1].Length++
			continue
		}
		runs = append(runs, ColorRun{Start: position, End: position, Length: 1, RGBA: pixel})
	}
	return runs
}
