package imagediff

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuggestImageOffsetReportsTranslationWithoutApplyingIt(t *testing.T) {
	reference := image.NewNRGBA(image.Rect(0, 0, 5, 3))
	actual := image.NewNRGBA(image.Rect(0, 0, 5, 3))
	reference.SetNRGBA(1, 1, color.NRGBA{R: 255, A: 255})
	actual.SetNRGBA(2, 1, color.NRGBA{R: 255, A: 255})

	offset := (&DecodedImages{Reference: reference, Actual: actual}).SuggestOffset(2, nil, nil)

	assert.Equal(t, -1, offset.X)
	assert.Equal(t, 0, offset.Y)
	assert.Zero(t, offset.RMSE)
	assert.Greater(t, offset.BaselineRMSE, offset.RMSE)
	assert.Equal(t, 1.0, offset.ImprovementRatio)
	assert.Equal(t, "candidate-translation", offset.Interpretation)
}

func TestSuggestImageOffsetKeepsUnrelatedDifferenceInconclusive(t *testing.T) {
	reference := image.NewNRGBA(image.Rect(0, 0, 5, 3))
	actual := image.NewNRGBA(image.Rect(0, 0, 5, 3))
	reference.SetNRGBA(1, 1, color.NRGBA{R: 255, A: 255})
	actual.SetNRGBA(1, 1, color.NRGBA{B: 255, A: 255})

	offset := (&DecodedImages{Reference: reference, Actual: actual}).SuggestOffset(2, nil, nil)

	assert.Equal(t, 2, absInt(offset.X)+absInt(offset.Y))
	assert.Equal(t, 1.0, offset.ImprovementRatio)
	assert.Equal(t, "inconclusive", offset.Interpretation)
}

func TestSuggestImageOffsetKeepsIdenticalTransparentImagesInconclusive(t *testing.T) {
	reference := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	actual := image.NewNRGBA(image.Rect(0, 0, 3, 3))

	offset := (&DecodedImages{Reference: reference, Actual: actual}).SuggestOffset(1, nil, nil)

	assert.Zero(t, offset.BaselineRMSE)
	assert.Zero(t, offset.ImprovementRatio)
	assert.Equal(t, "inconclusive", offset.Interpretation)
}
