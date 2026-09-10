package imagediff

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"sort"

	"github.com/cristianoliveira/pxp/internal/annotations"
	"github.com/cristianoliveira/pxp/internal/artifact"
)

type Bounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ignoredPixelMap struct {
	width  int
	height int
	pixels []bool
}

func newIgnoredPixelMap(width, height int, regions []Bounds) ignoredPixelMap {
	result := ignoredPixelMap{width: width, height: height}
	if len(regions) == 0 {
		return result
	}
	result.pixels = make([]bool, width*height)
	for _, region := range regions {
		startX, startY := max(0, region.X), max(0, region.Y)
		endX, endY := min(width, region.X+region.Width), min(height, region.Y+region.Height)
		for y := startY; y < endY; y++ {
			for x := startX; x < endX; x++ {
				result.pixels[y*width+x] = true
			}
		}
	}
	return result
}

func (m ignoredPixelMap) Contains(x, y int) bool {
	return len(m.pixels) > 0 && x >= 0 && x < m.width && y >= 0 && y < m.height && m.pixels[y*m.width+x]
}

type InputBounds struct {
	Reference Bounds `json:"reference"`
	Actual    Bounds `json:"actual"`
}

type Region struct {
	Bounds                  Bounds              `json:"bounds"`
	InputBounds             *InputBounds        `json:"inputBounds,omitempty"`
	ChangedPixels           int                 `json:"changedPixels"`
	ChangedRatio            float64             `json:"changedRatio"`
	RMSE                    float64             `json:"rmse"`
	EdgeRMSE                float64             `json:"edgeRmse"`
	PerceptualRMSE          float64             `json:"perceptualRmse"`
	PerceptualChangedPixels int                 `json:"perceptualChangedPixels"`
	PerceptualChangedRatio  float64             `json:"perceptualChangedRatio"`
	AntialiasedPixels       int                 `json:"antialiasedPixels"`
	DominantColorPairs      []ColorPair         `json:"dominantColorPairs,omitempty"`
	Classification          string              `json:"classification"`
	Annotations             []annotations.Match `json:"annotations,omitempty"`
}

type EvidenceBreakdown struct {
	RawOnlyPixels          int `json:"rawOnlyPixels"`
	PerceptualOnlyPixels   int `json:"perceptualOnlyPixels"`
	RawAndPerceptualPixels int `json:"rawAndPerceptualPixels"`
}

type ImageInput struct {
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Crop   *Bounds `json:"crop,omitempty"`
}

type ImageInputs struct {
	Reference ImageInput `json:"reference"`
	Actual    ImageInput `json:"actual"`
}

type ImageComparison struct {
	Width                   int               `json:"width"`
	Height                  int               `json:"height"`
	ChangedPixels           int               `json:"changedPixels"`
	ComparedPixels          int               `json:"comparedPixels"`
	ChangedRatio            float64           `json:"changedRatio"`
	RMSE                    float64           `json:"rmse"`
	RGBRMSE                 float64           `json:"rgbRmse"`
	LuminanceRMSE           float64           `json:"luminanceRmse"`
	AlphaRMSE               float64           `json:"alphaRmse"`
	EdgeRMSE                float64           `json:"edgeRmse"`
	PerceptualRMSE          float64           `json:"perceptualRmse"`
	PerceptualChangedPixels int               `json:"perceptualChangedPixels"`
	PerceptualChangedRatio  float64           `json:"perceptualChangedRatio"`
	PerceptualThreshold     float64           `json:"perceptualThreshold"`
	AntialiasedPixels       int               `json:"antialiasedPixels"`
	Evidence                EvidenceBreakdown `json:"evidence"`
	ComparedRegion          *Bounds           `json:"comparedRegion,omitempty"`
	Bounds                  *Bounds           `json:"bounds,omitempty"`
	ChangedRows             []int             `json:"changedRows,omitempty"`
	Regions                 []Region          `json:"regions,omitempty"`
	Mask                    string            `json:"mask,omitempty"`
	Overlay                 string            `json:"overlay,omitempty"`
	SuggestedOffset         *SuggestedOffset  `json:"suggestedOffset,omitempty"`
	MovedRegions            []RegionMovement  `json:"movedRegions,omitempty"`
	Inputs                  *ImageInputs      `json:"inputs,omitempty"`
}

func CompareImages(referencePath, actualPath, maskPath string, threshold uint8) (ImageComparison, error) {
	return CompareImagesInRegion(referencePath, actualPath, maskPath, threshold, nil)
}

func CompareImagesInRegion(referencePath, actualPath, maskPath string, threshold uint8, region *Bounds) (ImageComparison, error) {
	return CompareImagesWithIgnoredRegions(referencePath, actualPath, maskPath, threshold, region, nil)
}

func CompareImagesWithIgnoredRegions(referencePath, actualPath, maskPath string, threshold uint8, region *Bounds, ignored []Bounds) (ImageComparison, error) {
	return CompareImagesWithThresholds(referencePath, actualPath, maskPath, threshold, DefaultPerceptualThreshold, region, ignored)
}

func CompareImagesWithThresholds(referencePath, actualPath, maskPath string, threshold uint8, perceptualThreshold float64, region *Bounds, ignored []Bounds) (ImageComparison, error) {
	images, err := LoadDecodedImages(referencePath, actualPath)
	if err != nil {
		return ImageComparison{}, err
	}
	return images.Compare(maskPath, threshold, perceptualThreshold, region, ignored)
}

func (images *DecodedImages) Compare(maskPath string, threshold uint8, perceptualThreshold float64, region *Bounds, ignored []Bounds) (ImageComparison, error) {
	reference, actual := images.Reference, images.Actual

	imageWidth, imageHeight := reference.Bounds().Dx(), reference.Bounds().Dy()
	area := Bounds{Width: imageWidth, Height: imageHeight}
	if region != nil {
		area = *region
		if area.X < 0 || area.Y < 0 || area.Width <= 0 || area.Height <= 0 || area.X+area.Width > imageWidth || area.Y+area.Height > imageHeight {
			return ImageComparison{}, fmt.Errorf("region %d,%d,%d,%d is outside image bounds %dx%d", area.X, area.Y, area.Width, area.Height, imageWidth, imageHeight)
		}
	}
	width, height := area.Width, area.Height
	fullImage := Bounds{Width: imageWidth, Height: imageHeight}
	ignoredPixels := newIgnoredPixelMap(imageWidth, imageHeight, ignored)
	var mask *image.NRGBA
	if maskPath != "" {
		mask = image.NewNRGBA(image.Rect(0, 0, width, height))
	}
	changedPixels := make([]bool, width*height)
	changedRows := make([]bool, height)
	changed, perceptualChanged, antialiased, rawOnly, perceptualOnly, both, compared, minX, minY, maxX, maxY := 0, 0, 0, 0, 0, 0, 0, width, height, -1, -1
	var rgbSquaredError, luminanceSquaredError, alphaSquaredError, perceptualSquaredError float64
	hasTransparency, hasPixelDifference := false, false
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			absoluteX, absoluteY := area.X+x, area.Y+y
			if ignoredPixels.Contains(absoluteX, absoluteY) {
				continue
			}
			compared++
			r := reference.NRGBAAt(area.X+x, area.Y+y)
			a := actual.NRGBAAt(area.X+x, area.Y+y)
			hasTransparency = hasTransparency || r.A != 255 || a.A != 255
			if r == a {
				continue
			}
			hasPixelDifference = true
			delta := [4]uint8{absDiff(r.R, a.R), absDiff(r.G, a.G), absDiff(r.B, a.B), absDiff(r.A, a.A)}
			maxDelta := max(delta[0], delta[1], delta[2], delta[3])
			for _, value := range delta[:3] {
				rgbSquaredError += float64(value) * float64(value)
			}
			luminanceDelta := luminance(r) - luminance(a)
			luminanceSquaredError += luminanceDelta * luminanceDelta
			alphaSquaredError += float64(delta[3]) * float64(delta[3])
			perceptualDelta := perceptualColorDistance(r, a)
			perceptualSquaredError += perceptualDelta * perceptualDelta
			isPerceptualChange := perceptualDelta > perceptualThreshold
			if isPerceptualChange {
				perceptualChanged++
			}
			isRawChange := maxDelta > threshold
			switch {
			case isRawChange && isPerceptualChange:
				both++
			case isRawChange:
				rawOnly++
			case isPerceptualChange:
				perceptualOnly++
			}
			if !isRawChange {
				continue
			}
			changed++
			if likelyAntialiased(reference, actual, absoluteX, absoluteY, fullImage, ignoredPixels) {
				antialiased++
			}
			changedPixels[y*width+x] = true
			changedRows[y] = true
			minX, minY, maxX, maxY = min(minX, x), min(minY, y), max(maxX, x), max(maxY, y)
			if mask != nil {
				mask.SetNRGBA(x, y, color.NRGBA{R: 255, A: maxDelta})
			}
		}
	}
	if maskPath != "" {
		if err := encodePNG(maskPath, mask); err != nil {
			return ImageComparison{}, fmt.Errorf("write mask: %w", err)
		}
	}
	channelCount := 3
	squaredError := rgbSquaredError
	if hasTransparency {
		channelCount = 4
		squaredError += alphaSquaredError
	}
	result := ImageComparison{
		Width: width, Height: height, ChangedPixels: changed, ComparedPixels: compared, Mask: maskPath,
		PerceptualChangedPixels: perceptualChanged, PerceptualThreshold: perceptualThreshold, AntialiasedPixels: antialiased,
		Evidence: EvidenceBreakdown{RawOnlyPixels: rawOnly, PerceptualOnlyPixels: perceptualOnly, RawAndPerceptualPixels: both},
	}
	if compared > 0 {
		result.ChangedRatio = float64(changed) / float64(compared)
		result.RMSE = math.Sqrt(squaredError/float64(compared*channelCount)) / 255
		result.RGBRMSE = math.Sqrt(rgbSquaredError/float64(compared*3)) / 255
		result.LuminanceRMSE = math.Sqrt(luminanceSquaredError/float64(compared)) / 255
		result.AlphaRMSE = math.Sqrt(alphaSquaredError/float64(compared)) / 255
		if hasPixelDifference {
			result.EdgeRMSE = imageEdgeRMSE(reference, actual, area, ignoredPixels)
		}
		result.PerceptualRMSE = math.Sqrt(perceptualSquaredError / float64(compared))
		result.PerceptualChangedRatio = float64(perceptualChanged) / float64(compared)
	}
	if region != nil {
		comparedRegion := area
		result.ComparedRegion = &comparedRegion
	}
	if changed > 0 {
		result.Bounds = &Bounds{X: area.X + minX, Y: area.Y + minY, Width: maxX - minX + 1, Height: maxY - minY + 1}
		for row, hasChanges := range changedRows {
			if hasChanges {
				result.ChangedRows = append(result.ChangedRows, area.Y+row)
			}
		}
		result.Regions = findRegions(changedPixels, width, height)
		for index := range result.Regions {
			result.Regions[index].Bounds.X += area.X
			result.Regions[index].Bounds.Y += area.Y
		}
	}
	return result, nil
}

func PNGDimensions(path string) (int, int, error) {
	img, err := decodePNG(path)
	if err != nil {
		return 0, 0, err
	}
	return img.Bounds().Dx(), img.Bounds().Dy(), nil
}

func WriteCroppedPNG(inputPath, outputPath string, crop Bounds) error {
	img, err := decodePNG(inputPath)
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
	return encodePNG(outputPath, cropped)
}

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

func encodePNG(path string, img image.Image) error {
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

func imageEdgeRMSE(reference, actual *image.NRGBA, area Bounds, ignored ignoredPixelMap) float64 {
	var squaredError float64
	samples := 0
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			if ignored.Contains(x, y) {
				continue
			}
			referencePixel := reference.NRGBAAt(x, y)
			actualPixel := actual.NRGBAAt(x, y)
			for _, previous := range [][2]int{{x - 1, y}, {x, y - 1}} {
				if previous[0] < area.X || previous[1] < area.Y || ignored.Contains(previous[0], previous[1]) {
					continue
				}
				referencePrevious := reference.NRGBAAt(previous[0], previous[1])
				actualPrevious := actual.NRGBAAt(previous[0], previous[1])
				delta := (visibleLuminance(referencePixel) - visibleLuminance(referencePrevious)) - (visibleLuminance(actualPixel) - visibleLuminance(actualPrevious))
				squaredError += delta * delta
				samples++
			}
		}
	}
	if samples == 0 {
		return 0
	}
	return math.Sqrt(squaredError/float64(samples)) / 255
}

func visibleLuminance(pixel color.NRGBA) float64 {
	return luminance(pixel) * float64(pixel.A) / 255
}

func luminance(pixel color.NRGBA) float64 {
	return 0.2126*float64(pixel.R) + 0.7152*float64(pixel.G) + 0.0722*float64(pixel.B)
}

func findRegions(changed []bool, width, height int) []Region {
	visited := make([]bool, len(changed))
	regions := make([]Region, 0)
	for start := range changed {
		if !changed[start] || visited[start] {
			continue
		}
		queue := []int{start}
		visited[start] = true
		minX, maxX, minY, maxY, count := width, -1, height, -1, 0
		for len(queue) > 0 {
			index := queue[0]
			queue = queue[1:]
			x, y := index%width, index/width
			minX, maxX, minY, maxY, count = min(minX, x), max(maxX, x), min(minY, y), max(maxY, y), count+1
			for _, point := range [][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}} {
				nx, ny := point[0], point[1]
				if nx < 0 || nx >= width || ny < 0 || ny >= height {
					continue
				}
				next := ny*width + nx
				if changed[next] && !visited[next] {
					visited[next] = true
					queue = append(queue, next)
				}
			}
		}
		regions = append(regions, Region{Bounds: Bounds{X: minX, Y: minY, Width: maxX - minX + 1, Height: maxY - minY + 1}, ChangedPixels: count})
	}
	sort.SliceStable(regions, func(i, j int) bool { return regions[i].ChangedPixels > regions[j].ChangedPixels })
	return regions
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
