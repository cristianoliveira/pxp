// Package imageio owns image file decoding and artifact writing at the CLI boundary.
package imageio

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/cristianoliveira/pxp/internal/artifact"
	"github.com/cristianoliveira/pxp/internal/imagediff"
)

func LoadDecodedImages(referencePath, actualPath string) (*imagediff.DecodedImages, error) {
	type result struct {
		image *image.NRGBA
		err   error
	}
	load := func(path, label string) result {
		file, err := os.Open(path)
		if err != nil {
			return result{err: fmt.Errorf("decode %s: %w", label, err)}
		}
		decoded, decodeErr := png.Decode(file)
		closeErr := file.Close()
		if decodeErr != nil {
			return result{err: fmt.Errorf("decode %s: %w", label, decodeErr)}
		}
		if closeErr != nil {
			return result{err: fmt.Errorf("decode %s: %w", label, closeErr)}
		}
		bounds := decoded.Bounds()
		normalized := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		draw.Draw(normalized, normalized.Bounds(), decoded, bounds.Min, draw.Src)
		return result{image: normalized}
	}
	reference, actual := load(referencePath, "reference"), load(actualPath, "actual")
	if reference.err != nil {
		return nil, reference.err
	}
	if actual.err != nil {
		return nil, actual.err
	}
	if reference.image.Bounds().Size() != actual.image.Bounds().Size() {
		return nil, fmt.Errorf("image dimensions differ: reference is %dx%d, actual is %dx%d", reference.image.Bounds().Dx(), reference.image.Bounds().Dy(), actual.image.Bounds().Dx(), actual.image.Bounds().Dy())
	}
	return &imagediff.DecodedImages{Reference: reference.image, Actual: actual.image}, nil
}

func PNGDimensions(path string) (int, int, error) {
	img, err := decode(path)
	if err != nil {
		return 0, 0, err
	}
	return img.Bounds().Dx(), img.Bounds().Dy(), nil
}

func WritePNG(path string, img image.Image) error {
	file, err := artifact.CreateFile(path)
	if err != nil {
		return err
	}
	if err := png.Encode(file, img); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func WriteCroppedPNG(inputPath, outputPath string, crop imagediff.Bounds) error {
	img, err := decode(inputPath)
	if err != nil {
		return err
	}
	if crop.X < 0 || crop.Y < 0 || crop.Width <= 0 || crop.Height <= 0 || crop.X+crop.Width > img.Bounds().Dx() || crop.Y+crop.Height > img.Bounds().Dy() {
		return fmt.Errorf("crop %d,%d,%d,%d is outside image bounds %dx%d", crop.X, crop.Y, crop.Width, crop.Height, img.Bounds().Dx(), img.Bounds().Dy())
	}
	cropped := image.NewNRGBA(image.Rect(0, 0, crop.Width, crop.Height))
	for y := 0; y < crop.Height; y++ {
		for x := 0; x < crop.Width; x++ {
			cropped.Set(x, y, img.At(img.Bounds().Min.X+crop.X+x, img.Bounds().Min.Y+crop.Y+y))
		}
	}
	return WritePNG(outputPath, cropped)
}

func CompareImages(referencePath, actualPath, maskPath string, threshold uint8) (imagediff.ImageComparison, error) {
	return CompareImagesWithThresholds(referencePath, actualPath, maskPath, threshold, imagediff.DefaultPerceptualThreshold, nil, nil)
}

func CompareImagesInRegion(referencePath, actualPath, maskPath string, threshold uint8, region *imagediff.Bounds) (imagediff.ImageComparison, error) {
	return CompareImagesWithThresholds(referencePath, actualPath, maskPath, threshold, imagediff.DefaultPerceptualThreshold, region, nil)
}

func CompareImagesWithIgnoredRegions(referencePath, actualPath, maskPath string, threshold uint8, region *imagediff.Bounds, ignored []imagediff.Bounds) (imagediff.ImageComparison, error) {
	return CompareImagesWithThresholds(referencePath, actualPath, maskPath, threshold, imagediff.DefaultPerceptualThreshold, region, ignored)
}

func CompareImagesWithThresholds(referencePath, actualPath, maskPath string, threshold uint8, perceptualThreshold float64, region *imagediff.Bounds, ignored []imagediff.Bounds) (imagediff.ImageComparison, error) {
	images, err := LoadDecodedImages(referencePath, actualPath)
	if err != nil {
		return imagediff.ImageComparison{}, err
	}
	result, mask, err := images.Compare(threshold, perceptualThreshold, region, ignored)
	if err != nil {
		return imagediff.ImageComparison{}, err
	}
	if maskPath != "" {
		if err := WritePNG(maskPath, mask); err != nil {
			return imagediff.ImageComparison{}, fmt.Errorf("write mask: %w", err)
		}
		result.Mask = maskPath
	}
	return result, nil
}

func MeasureImageRegion(referencePath, actualPath string, bounds imagediff.Bounds, threshold uint8, ignored []imagediff.Bounds) (imagediff.RegionMetrics, error) {
	return MeasureImageRegionWithThresholds(referencePath, actualPath, bounds, threshold, imagediff.DefaultPerceptualThreshold, ignored)
}

func MeasureImageRegionWithThresholds(referencePath, actualPath string, bounds imagediff.Bounds, threshold uint8, perceptualThreshold float64, ignored []imagediff.Bounds) (imagediff.RegionMetrics, error) {
	images, err := LoadDecodedImages(referencePath, actualPath)
	if err != nil {
		return imagediff.RegionMetrics{}, err
	}
	metrics, err := images.MeasureRegions([]imagediff.Bounds{bounds}, threshold, perceptualThreshold, ignored)
	if err != nil {
		return imagediff.RegionMetrics{}, err
	}
	return metrics[0], nil
}

func MeasureImageRegionsWithThresholds(referencePath, actualPath string, regions []imagediff.Bounds, threshold uint8, perceptualThreshold float64, ignored []imagediff.Bounds) ([]imagediff.RegionMetrics, error) {
	images, err := LoadDecodedImages(referencePath, actualPath)
	if err != nil {
		return nil, err
	}
	return images.MeasureRegions(regions, threshold, perceptualThreshold, ignored)
}

func SuggestImageOffset(referencePath, actualPath string, radius int, region *imagediff.Bounds, ignored []imagediff.Bounds) (imagediff.SuggestedOffset, error) {
	images, err := LoadDecodedImages(referencePath, actualPath)
	if err != nil {
		return imagediff.SuggestedOffset{}, err
	}
	return images.SuggestOffset(radius, region, ignored), nil
}

func WriteImageOverlay(referencePath, actualPath, outputPath string, region *imagediff.Bounds, ignored []imagediff.Bounds) error {
	images, err := LoadDecodedImages(referencePath, actualPath)
	if err != nil {
		return err
	}
	overlay, err := images.Overlay(region, ignored)
	if err != nil {
		return err
	}
	return WritePNG(outputPath, overlay)
}

func IgnoredRegionsFromMask(maskPath, referencePath string) ([]imagediff.Bounds, error) {
	mask, err := decode(maskPath)
	if err != nil {
		return nil, fmt.Errorf("decode comparison mask: %w", err)
	}
	reference, err := decode(referencePath)
	if err != nil {
		return nil, fmt.Errorf("decode reference: %w", err)
	}
	if mask.Bounds().Size() != reference.Bounds().Size() {
		return nil, fmt.Errorf("comparison mask dimensions differ: mask is %dx%d, reference is %dx%d", mask.Bounds().Dx(), mask.Bounds().Dy(), reference.Bounds().Dx(), reference.Bounds().Dy())
	}
	regions := make([]imagediff.Bounds, 0)
	for y := 0; y < mask.Bounds().Dy(); y++ {
		runStart := -1
		for x := 0; x <= mask.Bounds().Dx(); x++ {
			excluded := false
			if x < mask.Bounds().Dx() {
				pixel := color.NRGBAModel.Convert(mask.At(mask.Bounds().Min.X+x, mask.Bounds().Min.Y+y)).(color.NRGBA)
				excluded = pixel.A == 0 || pixel.R == 0 && pixel.G == 0 && pixel.B == 0
			}
			if excluded && runStart < 0 {
				runStart = x
			}
			if !excluded && runStart >= 0 {
				regions = append(regions, imagediff.Bounds{X: runStart, Y: y, Width: x - runStart, Height: 1})
				runStart = -1
			}
		}
	}
	return regions, nil
}

func decode(path string) (image.Image, error) {
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
