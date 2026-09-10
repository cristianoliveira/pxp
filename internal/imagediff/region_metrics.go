package imagediff

import (
	"fmt"
	"image"
	"math"
	"sort"
)

type ColorPair struct {
	Reference string `json:"reference"`
	Actual    string `json:"actual"`
	Pixels    int    `json:"pixels"`
}

type RegionMetrics struct {
	ChangedPixels           int         `json:"changedPixels"`
	ChangedRatio            float64     `json:"changedRatio"`
	RMSE                    float64     `json:"rmse"`
	EdgeRMSE                float64     `json:"edgeRmse"`
	PerceptualRMSE          float64     `json:"perceptualRmse"`
	PerceptualChangedPixels int         `json:"perceptualChangedPixels"`
	PerceptualChangedRatio  float64     `json:"perceptualChangedRatio"`
	AntialiasedPixels       int         `json:"antialiasedPixels"`
	DominantColorPairs      []ColorPair `json:"dominantColorPairs,omitempty"`
}

func (images *DecodedImages) MeasureRegions(
	regions []Bounds,
	threshold uint8,
	perceptualThreshold float64,
	ignored []Bounds,
) ([]RegionMetrics, error) {
	metrics := make([]RegionMetrics, len(regions))
	var err error
	for index, bounds := range regions {
		metrics[index], err = measureImageRegion(
			images.Reference,
			images.Actual,
			bounds,
			threshold,
			perceptualThreshold,
			ignored,
		)
		if err != nil {
			return nil, err
		}
	}
	return metrics, nil
}

func measureImageRegion(
	reference, actual *image.NRGBA,
	bounds Bounds,
	threshold uint8,
	perceptualThreshold float64,
	ignored []Bounds,
) (RegionMetrics, error) {
	if bounds.X < 0 || bounds.Y < 0 || bounds.Width <= 0 || bounds.Height <= 0 ||
		bounds.X+bounds.Width > reference.Bounds().Dx() ||
		bounds.Y+bounds.Height > reference.Bounds().Dy() {
		return RegionMetrics{}, fmt.Errorf(
			"region %d,%d,%d,%d is outside image bounds %dx%d",
			bounds.X,
			bounds.Y,
			bounds.Width,
			bounds.Height,
			reference.Bounds().Dx(),
			reference.Bounds().Dy(),
		)
	}
	fullImage := Bounds{Width: reference.Bounds().Dx(), Height: reference.Bounds().Dy()}
	ignoredPixels := newIgnoredPixelMap(fullImage.Width, fullImage.Height, ignored)
	changed, perceptualChanged, antialiased, compared, channels := 0, 0, 0, 0, 3
	colorPairs := make(map[[8]uint8]int)
	var rgbError, alphaError, perceptualError float64
	transparent, hasPixelDifference := false, false
	for y := bounds.Y; y < bounds.Y+bounds.Height; y++ {
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			if ignoredPixels.Contains(x, y) {
				continue
			}
			compared++
			r := reference.NRGBAAt(x, y)
			a := actual.NRGBAAt(x, y)
			transparent = transparent || r.A != 255 || a.A != 255
			if r == a {
				continue
			}
			hasPixelDifference = true
			deltas := []uint8{
				absDiff(r.R, a.R),
				absDiff(r.G, a.G),
				absDiff(r.B, a.B),
				absDiff(r.A, a.A),
			}
			if max(deltas[0], deltas[1], deltas[2], deltas[3]) > threshold {
				changed++
				if likelyAntialiased(reference, actual, x, y, fullImage, ignoredPixels) {
					antialiased++
				}
				colorPairs[[8]uint8{r.R, r.G, r.B, r.A, a.R, a.G, a.B, a.A}]++
			}
			for _, d := range deltas[:3] {
				rgbError += float64(d) * float64(d)
			}
			alphaError += float64(deltas[3]) * float64(deltas[3])
			perceptualDelta := perceptualColorDistance(r, a)
			perceptualError += perceptualDelta * perceptualDelta
			if perceptualDelta > perceptualThreshold {
				perceptualChanged++
			}
		}
	}
	if compared == 0 {
		return RegionMetrics{}, nil
	}
	total := rgbError
	if transparent {
		channels = 4
		total += alphaError
	}
	edgeRMSE := 0.0
	if hasPixelDifference {
		edgeRMSE = imageEdgeRMSE(reference, actual, bounds, ignoredPixels)
	}
	return RegionMetrics{
		ChangedPixels: changed, ChangedRatio: float64(changed) / float64(compared),
		RMSE: math.Sqrt(total/float64(compared*channels)) / 255, EdgeRMSE: edgeRMSE,
		PerceptualRMSE: math.Sqrt(
			perceptualError / float64(compared),
		), PerceptualChangedPixels: perceptualChanged,
		PerceptualChangedRatio: float64(
			perceptualChanged,
		) / float64(
			compared,
		), AntialiasedPixels: antialiased,
		DominantColorPairs: dominantColorPairs(colorPairs),
	}, nil
}

func dominantColorPairs(counts map[[8]uint8]int) []ColorPair {
	pairs := make([]ColorPair, 0, len(counts))
	for colors, pixels := range counts {
		pairs = append(
			pairs,
			ColorPair{
				Reference: formatPixelColor(colors[0:4]),
				Actual:    formatPixelColor(colors[4:8]),
				Pixels:    pixels,
			},
		)
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Pixels != pairs[j].Pixels {
			return pairs[i].Pixels > pairs[j].Pixels
		}
		if pairs[i].Reference != pairs[j].Reference {
			return pairs[i].Reference < pairs[j].Reference
		}
		return pairs[i].Actual < pairs[j].Actual
	})
	if len(pairs) > 3 {
		return pairs[:3]
	}
	return pairs
}

func formatPixelColor(channels []uint8) string {
	if channels[3] == 255 {
		return fmt.Sprintf("#%02X%02X%02X", channels[0], channels[1], channels[2])
	}
	return fmt.Sprintf("#%02X%02X%02X%02X", channels[0], channels[1], channels[2], channels[3])
}
