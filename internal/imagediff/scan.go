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

// ScanLine contains one source row or column and its exact color runs.
type ScanLine struct {
	Index int
	Runs  []ColorRun
}

// Scan returns color runs for the requested row or column in both images.
func (images *DecodedImages) Scan(axis ScanAxis, index int) ([]ColorRun, []ColorRun, error) {
	reference, actual, err := images.ScanBand(axis, index, index)
	if err != nil {
		return nil, nil, err
	}
	return reference[0].Runs, actual[0].Runs, nil
}

// ScanBand returns one exact run list per inclusive row or column in both images.
// It never averages or combines pixels across source lines.
func (images *DecodedImages) ScanBand(
	axis ScanAxis, start, end int,
) ([]ScanLine, []ScanLine, error) {
	if err := validateMeasurementImages(images); err != nil {
		return nil, nil, err
	}
	if axis != ScanHorizontal && axis != ScanVertical {
		return nil, nil, fmt.Errorf("unsupported scan axis %q", axis)
	}
	if start > end {
		return nil, nil, fmt.Errorf("scan range start %d is greater than end %d", start, end)
	}
	bounds := images.Reference.Bounds()
	limit, length := scanDimensions(bounds, axis)
	if start < 0 || end >= limit {
		point := image.Point{X: 0, Y: start}
		if axis == ScanVertical {
			point = image.Point{X: start, Y: 0}
		}
		return nil, nil, &MeasurementBoundsError{Point: point, Size: bounds.Size()}
	}
	reference := make([]ScanLine, 0, end-start+1)
	actual := make([]ScanLine, 0, end-start+1)
	for index := start; index <= end; index++ {
		reference = append(reference, ScanLine{
			Index: index, Runs: scanRuns(images.Reference, axis, index, length),
		})
		actual = append(actual, ScanLine{
			Index: index, Runs: scanRuns(images.Actual, axis, index, length),
		})
	}
	return reference, actual, nil
}

func scanDimensions(bounds image.Rectangle, axis ScanAxis) (limit, length int) {
	limit = bounds.Dy()
	length = bounds.Dx()
	if axis == ScanVertical {
		limit = bounds.Dx()
		length = bounds.Dy()
	}
	return limit, length
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
