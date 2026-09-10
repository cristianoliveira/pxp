package imagediff

import (
	"image/color"
	"math"
	"sort"
)

const (
	candidateTranslationImprovementRatio = 0.10
	candidateTranslationMinimumOverlap   = 0.70
)

const (
	offsetInterpretationInconclusive         = "inconclusive"
	offsetInterpretationCandidateTranslation = "candidate-translation"
)

type SuggestedOffset struct {
	X                int     `json:"x"`
	Y                int     `json:"y"`
	RMSE             float64 `json:"rmse"`
	BaselineRMSE     float64 `json:"baselineRmse"`
	ImprovementRatio float64 `json:"improvementRatio"`
	Interpretation   string  `json:"interpretation"`
}

func (images *DecodedImages) SuggestOffset(radius int, region *Bounds, ignored []Bounds) SuggestedOffset {
	reference, actual := images.Reference, images.Actual
	area := Bounds{Width: reference.Bounds().Dx(), Height: reference.Bounds().Dy()}
	if region != nil {
		area = *region
	}
	candidates := make([]SuggestedOffset, 0, (radius*2+1)*(radius*2+1))
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			candidates = append(candidates, SuggestedOffset{X: x, Y: y})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		ai, aj := absInt(candidates[i].X)+absInt(candidates[i].Y), absInt(candidates[j].X)+absInt(candidates[j].Y)
		if ai != aj {
			return ai < aj
		}
		if candidates[i].Y != candidates[j].Y {
			return candidates[i].Y < candidates[j].Y
		}
		return candidates[i].X < candidates[j].X
	})
	ignoredPixels := newIgnoredPixelMap(reference.Bounds().Dx(), reference.Bounds().Dy(), ignored)
	best := SuggestedOffset{RMSE: math.Inf(1)}
	baselineRMSE := math.Inf(1)
	for _, candidate := range candidates {
		candidate.RMSE = offsetRMSE(reference, actual, area, ignoredPixels, candidate.X, candidate.Y)
		if candidate.X == 0 && candidate.Y == 0 {
			baselineRMSE = candidate.RMSE
		}
		if candidate.RMSE < best.RMSE {
			best = candidate
		}
	}
	return interpretSuggestedOffset(best, baselineRMSE, area)
}

func interpretSuggestedOffset(best SuggestedOffset, baselineRMSE float64, area Bounds) SuggestedOffset {
	best.BaselineRMSE = baselineRMSE
	best.Interpretation = offsetInterpretationInconclusive
	if baselineRMSE <= 0 || math.IsInf(baselineRMSE, 0) || math.IsNaN(baselineRMSE) {
		return best
	}

	best.ImprovementRatio = (baselineRMSE - best.RMSE) / baselineRMSE
	overlapWidth := max(0, area.Width-absInt(best.X))
	overlapHeight := max(0, area.Height-absInt(best.Y))
	overlapRatio := float64(overlapWidth*overlapHeight) / float64(area.Width*area.Height)
	if (best.X != 0 || best.Y != 0) &&
		best.ImprovementRatio >= candidateTranslationImprovementRatio &&
		overlapRatio >= candidateTranslationMinimumOverlap {
		best.Interpretation = offsetInterpretationCandidateTranslation
	}
	return best
}

func offsetRMSE(reference, actual interface{ At(int, int) color.Color }, area Bounds, ignored ignoredPixelMap, offsetX, offsetY int) float64 {
	var rgbError, alphaError float64
	compared := 0
	transparent := false
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			actualX, actualY := x-offsetX, y-offsetY
			if actualX < area.X || actualX >= area.X+area.Width || actualY < area.Y || actualY >= area.Y+area.Height || ignored.Contains(x, y) {
				continue
			}
			r := color.NRGBAModel.Convert(reference.At(x, y)).(color.NRGBA)
			a := color.NRGBAModel.Convert(actual.At(actualX, actualY)).(color.NRGBA)
			for _, d := range []uint8{absDiff(r.R, a.R), absDiff(r.G, a.G), absDiff(r.B, a.B)} {
				rgbError += float64(d) * float64(d)
			}
			d := absDiff(r.A, a.A)
			alphaError += float64(d) * float64(d)
			transparent = transparent || r.A != 255 || a.A != 255
			compared++
		}
	}
	if compared == 0 {
		return math.Inf(1)
	}
	channels := 3
	total := rgbError
	if transparent {
		channels = 4
		total += alphaError
	}
	return math.Sqrt(total/float64(compared*channels)) / 255
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
