package imagediff

import "sort"

const (
	candidateRegionMovementMinimumConfidence = 0.50
	maximumReportedRegionMovements           = 5
)

// RegionMovement is advisory evidence that content inside one local search area
// appears translated. It never aligns images or changes comparison metrics.
type RegionMovement struct {
	Bounds     Bounds  `json:"bounds"`
	DX         int     `json:"dx"`
	DY         int     `json:"dy"`
	Confidence float64 `json:"confidence"`
}

func (images *DecodedImages) SuggestRegionMovements(
	regions []Bounds,
	radius int,
	ignored []Bounds,
) []RegionMovement {
	if radius <= 0 || len(regions) == 0 {
		return nil
	}

	imageBounds := Bounds{
		Width:  images.Reference.Bounds().Dx(),
		Height: images.Reference.Bounds().Dy(),
	}
	movements := make([]RegionMovement, 0, len(regions))
	for _, region := range regions {
		searchBounds := expandAndClipBounds(region, radius, imageBounds)
		offset := images.SuggestOffset(radius, &searchBounds, ignored)
		if offset.Interpretation != offsetInterpretationCandidateTranslation ||
			offset.ImprovementRatio < candidateRegionMovementMinimumConfidence {
			continue
		}

		movement := RegionMovement{
			Bounds:     searchBounds,
			DX:         -offset.X,
			DY:         -offset.Y,
			Confidence: offset.ImprovementRatio,
		}
		movements = appendMovementEvidence(movements, movement)
	}

	sort.SliceStable(movements, func(i, j int) bool {
		if movements[i].Confidence != movements[j].Confidence {
			return movements[i].Confidence > movements[j].Confidence
		}
		if movements[i].Bounds.Y != movements[j].Bounds.Y {
			return movements[i].Bounds.Y < movements[j].Bounds.Y
		}
		return movements[i].Bounds.X < movements[j].Bounds.X
	})
	if len(movements) > maximumReportedRegionMovements {
		return movements[:maximumReportedRegionMovements]
	}
	return movements
}

func expandAndClipBounds(bounds Bounds, padding int, limit Bounds) Bounds {
	left := max(limit.X, bounds.X-padding)
	top := max(limit.Y, bounds.Y-padding)
	right := min(limit.X+limit.Width, bounds.X+bounds.Width+padding)
	bottom := min(limit.Y+limit.Height, bounds.Y+bounds.Height+padding)
	return Bounds{X: left, Y: top, Width: right - left, Height: bottom - top}
}

func appendMovementEvidence(movements []RegionMovement, candidate RegionMovement) []RegionMovement {
	for index, movement := range movements {
		if movement.DX != candidate.DX || movement.DY != candidate.DY ||
			!boundsOverlap(movement.Bounds, candidate.Bounds) {
			continue
		}
		if candidate.Confidence > movement.Confidence {
			movements[index] = candidate
		}
		return movements
	}
	return append(movements, candidate)
}

func boundsOverlap(first, second Bounds) bool {
	return first.X < second.X+second.Width && second.X < first.X+first.Width &&
		first.Y < second.Y+second.Height && second.Y < first.Y+first.Height
}
