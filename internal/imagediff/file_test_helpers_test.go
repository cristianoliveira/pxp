package imagediff

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"

	"github.com/cristianoliveira/pxp/internal/artifact"
)

func decodePNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	img, decodeErr := png.Decode(file)
	closeErr := file.Close()
	if decodeErr != nil {
		return nil, decodeErr
	}
	return img, closeErr
}

func decodeNRGBA(path string) (*image.NRGBA, error) {
	decoded, err := decodePNG(path)
	if err != nil {
		return nil, err
	}
	bounds := decoded.Bounds()
	normalized := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(normalized, normalized.Bounds(), decoded, bounds.Min, draw.Src)
	return normalized, nil
}

func LoadDecodedImages(referencePath, actualPath string) (*DecodedImages, error) {
	reference, err := decodeNRGBA(referencePath)
	if err != nil {
		return nil, err
	}
	actual, err := decodeNRGBA(actualPath)
	if err != nil {
		return nil, err
	}
	if reference.Bounds().Size() != actual.Bounds().Size() {
		return nil, imageDimensionsError(reference, actual)
	}
	return &DecodedImages{Reference: reference, Actual: actual}, nil
}

func imageDimensionsError(reference, actual *image.NRGBA) error {
	return &dimensionsError{reference: reference.Bounds().Size(), actual: actual.Bounds().Size()}
}

type dimensionsError struct{ reference, actual image.Point }

func (e *dimensionsError) Error() string {
	return fmt.Sprintf("image dimensions differ: reference is %dx%d, actual is %dx%d", e.reference.X, e.reference.Y, e.actual.X, e.actual.Y)
}

func CompareImages(referencePath, actualPath, maskPath string, threshold uint8) (ImageComparison, error) {
	return CompareImagesWithThresholds(referencePath, actualPath, maskPath, threshold, DefaultPerceptualThreshold, nil, nil)
}
func CompareImagesInRegion(referencePath, actualPath, maskPath string, threshold uint8, region *Bounds) (ImageComparison, error) {
	return CompareImagesWithThresholds(referencePath, actualPath, maskPath, threshold, DefaultPerceptualThreshold, region, nil)
}
func CompareImagesWithIgnoredRegions(referencePath, actualPath, maskPath string, threshold uint8, region *Bounds, ignored []Bounds) (ImageComparison, error) {
	return CompareImagesWithThresholds(referencePath, actualPath, maskPath, threshold, DefaultPerceptualThreshold, region, ignored)
}
func CompareImagesWithThresholds(referencePath, actualPath, maskPath string, threshold uint8, perceptualThreshold float64, region *Bounds, ignored []Bounds) (ImageComparison, error) {
	images, err := LoadDecodedImages(referencePath, actualPath)
	if err != nil {
		return ImageComparison{}, err
	}
	result, mask, err := images.Compare(threshold, perceptualThreshold, region, ignored)
	if err != nil {
		return ImageComparison{}, err
	}
	if maskPath != "" {
		file, err := artifact.CreateFile(maskPath)
		if err != nil {
			return ImageComparison{}, err
		}
		err = png.Encode(file, mask)
		_ = file.Close()
		if err != nil {
			return ImageComparison{}, err
		}
		result.Mask = maskPath
	}
	return result, nil
}
func MeasureImageRegion(referencePath, actualPath string, bounds Bounds, threshold uint8, ignored []Bounds) (RegionMetrics, error) {
	return MeasureImageRegionWithThresholds(referencePath, actualPath, bounds, threshold, DefaultPerceptualThreshold, ignored)
}
func MeasureImageRegionWithThresholds(referencePath, actualPath string, bounds Bounds, threshold uint8, perceptualThreshold float64, ignored []Bounds) (RegionMetrics, error) {
	images, err := LoadDecodedImages(referencePath, actualPath)
	if err != nil {
		return RegionMetrics{}, err
	}
	metrics, err := images.MeasureRegions([]Bounds{bounds}, threshold, perceptualThreshold, ignored)
	if err != nil {
		return RegionMetrics{}, err
	}
	return metrics[0], nil
}
func MeasureImageRegionsWithThresholds(referencePath, actualPath string, regions []Bounds, threshold uint8, perceptualThreshold float64, ignored []Bounds) ([]RegionMetrics, error) {
	images, err := LoadDecodedImages(referencePath, actualPath)
	if err != nil {
		return nil, err
	}
	return images.MeasureRegions(regions, threshold, perceptualThreshold, ignored)
}
func SuggestImageOffset(referencePath, actualPath string, radius int, region *Bounds, ignored []Bounds) (SuggestedOffset, error) {
	images, err := LoadDecodedImages(referencePath, actualPath)
	if err != nil {
		return SuggestedOffset{}, err
	}
	return images.SuggestOffset(radius, region, ignored), nil
}
func WriteImageOverlay(referencePath, actualPath, outputPath string, region *Bounds, ignored []Bounds) error {
	images, err := LoadDecodedImages(referencePath, actualPath)
	if err != nil {
		return err
	}
	overlay, err := images.Overlay(region, ignored)
	if err != nil {
		return err
	}
	file, err := artifact.CreateFile(outputPath)
	if err != nil {
		return err
	}
	err = png.Encode(file, overlay)
	_ = file.Close()
	return err
}
func IgnoredRegionsFromMask(maskPath, referencePath string) ([]Bounds, error) {
	mask, err := decodePNG(maskPath)
	if err != nil {
		return nil, err
	}
	reference, err := decodePNG(referencePath)
	if err != nil {
		return nil, err
	}
	if mask.Bounds().Size() != reference.Bounds().Size() {
		return nil, imageDimensionsError(image.NewNRGBA(reference.Bounds()), image.NewNRGBA(mask.Bounds()))
	}
	regions := []Bounds{}
	for y := 0; y < mask.Bounds().Dy(); y++ {
		run := -1
		for x := 0; x <= mask.Bounds().Dx(); x++ {
			excluded := false
			if x < mask.Bounds().Dx() {
				p := mask.At(mask.Bounds().Min.X+x, mask.Bounds().Min.Y+y)
				r, g, b, a := p.RGBA()
				excluded = a == 0 || r == 0 && g == 0 && b == 0
			}
			if excluded && run < 0 {
				run = x
			}
			if !excluded && run >= 0 {
				regions = append(regions, Bounds{X: run, Y: y, Width: x - run, Height: 1})
				run = -1
			}
		}
	}
	return regions, nil
}
