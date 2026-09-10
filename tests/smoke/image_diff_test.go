package smoke

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPixelPerfectStandaloneCLI(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	mask := filepath.Join(t.TempDir(), "mask.png")
	output, err := pixelPerfectCommand(binary, filepath.Join(fixtures, "reference.png"), filepath.Join(fixtures, "two-regions.png"), "--output", mask).CombinedOutput()

	require.NoError(t, err, string(output))
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal(output, &comparison))
	assert.Equal(t, 2, comparison.ChangedPixels)
	assert.Equal(t, &diff.Bounds{X: 0, Y: 0, Width: 4, Height: 3}, comparison.Bounds)
	assert.Equal(t, []int{0, 2}, comparison.ChangedRows)
	require.Len(t, comparison.Regions, 2)
	assert.NotEmpty(t, comparison.Regions[0].Classification)
}

func TestPixelPerfectReportsAdvisoryRegionMovement(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	mask := filepath.Join(t.TempDir(), "mask.png")
	output, err := pixelPerfectCommand(binary,
		filepath.Join(fixtures, "offset-reference.png"),
		filepath.Join(fixtures, "offset-right-one.png"),
		"--output", mask,
		"--suggest-movement", "2",
	).CombinedOutput()

	require.NoError(t, err, string(output))
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal(output, &comparison))
	require.Len(t, comparison.MovedRegions, 1)
	assert.Equal(t, 1, comparison.MovedRegions[0].DX)
	assert.Zero(t, comparison.MovedRegions[0].DY)
	assert.Greater(t, comparison.MovedRegions[0].Confidence, 0.0)
}

func TestPixelPerfectCreatesMissingArtifactDirectories(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	dir := t.TempDir()
	mask := filepath.Join(dir, "missing", "masks", "diff.png")
	overlay := filepath.Join(dir, "missing", "overlays", "diff.png")
	report := filepath.Join(dir, "missing", "reports", "diff.html")

	output, err := pixelPerfectCommand(binary,
		filepath.Join(fixtures, "reference.png"),
		filepath.Join(fixtures, "two-regions.png"),
		"--output", mask,
		"--overlay", overlay,
		"--report", report,
	).CombinedOutput()

	require.NoError(t, err, string(output))
	assert.FileExists(t, mask)
	assert.FileExists(t, overlay)
	assert.FileExists(t, report)
}

func TestPixelPerfectProbeCLI(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	output, err := pixelPerfectCommand(binary, "probe", filepath.Join(fixtures, "reference.png"), filepath.Join(fixtures, "two-regions.png"), "--at", "0,0").CombinedOutput()

	require.NoError(t, err, string(output))
	assert.Equal(t, "x,y,ref,act,delta,input_ref,input_act\n0,0,#FFFFFF,#000000,255,,\n", string(output))
}

func TestPixelPerfectProbeLineCLI(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	output, err := pixelPerfectCommand(binary, "probe", filepath.Join(fixtures, "reference.png"), filepath.Join(fixtures, "two-regions.png"), "--from", "0,0", "--to", "3,2", "--step", "2").CombinedOutput()

	require.NoError(t, err, string(output))
	assert.Contains(t, string(output), "0,0,#FFFFFF,#000000,255,,")
	assert.Contains(t, string(output), "3,2,#FFFFFF,#FF0000,255,,")
}

func TestPixelPerfectScanCLI(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	output, err := pixelPerfectCommand(binary, "scan", filepath.Join(fixtures, "reference.png"), filepath.Join(fixtures, "two-regions.png"), "--y", "0").CombinedOutput()

	require.NoError(t, err, string(output))
	assert.Contains(t, string(output), "image,axis,index,start,end,length,hex,input_axis,input_index")
	assert.Contains(t, string(output), "ref,x,0,0,3,4,#FFFFFF,,")
	assert.Contains(t, string(output), "act,x,0,0,0,1,#000000,,")
}

func TestPixelPerfectScanCLIErrorContracts(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	output, err := pixelPerfectCommand(binary, "scan", filepath.Join(fixtures, "reference.png"), filepath.Join(fixtures, "two-regions.png"), "--x", "0", "--y", "0").CombinedOutput()

	require.Error(t, err)
	assert.Contains(t, string(output), "provide exactly one of --x/--column or --y/--row")
}

func TestPixelPerfectProbeCLIErrorContracts(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	output, err := pixelPerfectCommand(binary, "probe", filepath.Join(fixtures, "reference.png"), filepath.Join(fixtures, "unequal-dimensions.png"), "--at", "0,0").CombinedOutput()

	require.Error(t, err)
	assert.Contains(t, string(output), `"message": "Command could not complete."`)
}

func TestPixelPerfectStandaloneCLIDefaultMask(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	workDir := t.TempDir()
	reference := filepath.Join(workDir, "reference.png")
	actual := filepath.Join(workDir, "actual.png")
	require.NoError(t, copyFile(filepath.Join(fixtures, "reference.png"), reference))
	require.NoError(t, copyFile(filepath.Join(fixtures, "two-regions.png"), actual))
	output, err := pixelPerfectCommand(binary, reference, actual, "--threshold", "8").CombinedOutput()

	require.NoError(t, err, string(output))
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal(output, &comparison))
	assert.Equal(t, 2, comparison.ChangedPixels)
	assert.Equal(t, filepath.Join(workDir, "actual.diff.png"), comparison.Mask)
	_, statErr := os.Stat(comparison.Mask)
	require.NoError(t, statErr)
}

func TestImageDiffScenarios(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	tests := []struct {
		name                  string
		actual                string
		flags                 []string
		changed               int
		regionCount           int
		expectedCompared      int
		expectsOffset         bool
		expectsColorPair      bool
		expectsClassification bool
	}{
		{name: "identical images", actual: "identical.png"},
		{name: "disconnected changes", actual: "two-regions.png", changed: 2, regionCount: 2},
		{name: "region reports dominant color pair", actual: "two-regions.png", changed: 2, regionCount: 2, expectsColorPair: true},
		{name: "region reports likely mismatch class", actual: "two-regions.png", changed: 2, regionCount: 2, expectsClassification: true},
		{name: "tiny regions can be omitted", actual: "two-regions.png", flags: []string{"--min-region-pixels", "2"}, changed: 2},
		{name: "nearby regions can be grouped", actual: "two-regions.png", flags: []string{"--region-gap", "4"}, changed: 2, regionCount: 1},
		{name: "known dynamic area can be ignored", actual: "two-regions.png", flags: []string{"--ignore-region", "0,0,1,1"}, changed: 1, regionCount: 1},
		{name: "comparison mask selects pixels", actual: "two-regions.png", flags: []string{"--mask", filepath.Join(fixtures, "comparison-mask.png")}, changed: 1, regionCount: 1, expectedCompared: 11},
		{name: "translation can be reported", actual: "two-regions.png", flags: []string{"--suggest-offset", "1"}, changed: 2, regionCount: 2, expectsOffset: true},
		{name: "threshold ignores subtle rendering noise", actual: "subtle-change.png", flags: []string{"--threshold", "5"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outputDir := t.TempDir()
			mask := filepath.Join(outputDir, "mask.png")
			overlay := filepath.Join(outputDir, "overlay.png")
			args := []string{filepath.Join(fixtures, "reference.png"), filepath.Join(fixtures, test.actual), "--output", mask, "--overlay", overlay}
			args = append(args, test.flags...)
			output, err := pixelPerfectCommand(binary, args...).CombinedOutput()

			require.NoError(t, err, string(output))
			var comparison diff.ImageComparison
			require.NoError(t, json.Unmarshal(output, &comparison))
			assert.Equal(t, test.changed, comparison.ChangedPixels)
			assert.Len(t, comparison.Regions, test.regionCount)
			if test.regionCount > 0 {
				assert.Greater(t, comparison.Regions[0].ChangedRatio, 0.0)
				assert.Greater(t, comparison.Regions[0].RMSE, 0.0)
				assert.Greater(t, comparison.Regions[0].PerceptualRMSE, 0.0)
			}
			if test.expectedCompared > 0 {
				assert.Equal(t, test.expectedCompared, comparison.ComparedPixels)
			}
			if test.expectsOffset {
				assert.NotNil(t, comparison.SuggestedOffset)
			}
			if test.expectsColorPair {
				require.NotEmpty(t, comparison.Regions[0].DominantColorPairs)
				assert.Greater(t, comparison.Regions[0].DominantColorPairs[0].Pixels, 0)
			}
			if test.expectsClassification {
				assert.NotEmpty(t, comparison.Regions[0].Classification)
			}
			_, err = os.Stat(mask)
			require.NoError(t, err)
			_, err = os.Stat(overlay)
			require.NoError(t, err)
		})
	}
}

func TestPixelPerfectBoundaryAndCompositingScenarios(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	tests := []struct {
		name, reference, actual string
		flags                   []string
		changed, compared       int
		assertResult            func(*testing.T, diff.ImageComparison)
	}{
		{name: "difference equal to threshold is ignored", reference: "reference.png", actual: "threshold-equal.png", flags: []string{"--threshold", "5"}, compared: 12},
		{name: "difference above threshold is detected", reference: "reference.png", actual: "threshold-exceeded.png", flags: []string{"--threshold", "5"}, changed: 12, compared: 12},
		{name: "alpha-only changes affect alpha metric", reference: "reference.png", actual: "alpha-only-change.png", changed: 12, compared: 12, assertResult: func(t *testing.T, result diff.ImageComparison) { assert.Greater(t, result.AlphaRMSE, 0.0) }},
		{name: "hidden RGB is ignored for transparent pixels", reference: "all-excluded-mask.png", actual: "transparent-hidden-rgb.png", compared: 12},
		{name: "fully excluded comparison is valid", reference: "reference.png", actual: "two-regions.png", flags: []string{"--mask", filepath.Join(fixtures, "all-excluded-mask.png")}},
		{name: "translation direction is exact", reference: "offset-reference.png", actual: "offset-right-one.png", flags: []string{"--suggest-offset", "2"}, changed: 2, compared: 12, assertResult: func(t *testing.T, result diff.ImageComparison) {
			require.NotNil(t, result.SuggestedOffset)
			assert.Equal(t, -1, result.SuggestedOffset.X)
			assert.Equal(t, 0, result.SuggestedOffset.Y)
			assert.Greater(t, result.SuggestedOffset.BaselineRMSE, result.SuggestedOffset.RMSE)
			assert.Equal(t, 1.0, result.SuggestedOffset.ImprovementRatio)
			assert.Equal(t, "candidate-translation", result.SuggestedOffset.Interpretation)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mask := filepath.Join(t.TempDir(), "mask.png")
			args := []string{filepath.Join(fixtures, test.reference), filepath.Join(fixtures, test.actual), "--output", mask}
			args = append(args, test.flags...)
			output, err := pixelPerfectCommand(binary, args...).CombinedOutput()
			require.NoError(t, err, string(output))
			var comparison diff.ImageComparison
			require.NoError(t, json.Unmarshal(output, &comparison))
			assert.Equal(t, test.changed, comparison.ChangedPixels)
			assert.Equal(t, test.compared, comparison.ComparedPixels)
			assertPNGDimensions(t, mask, 4, 3)
			if test.assertResult != nil {
				test.assertResult(t, comparison)
			}
		})
	}
}

func TestPixelPerfectAnnotationsEnrichRegionsWithoutChangingMetrics(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	dir := t.TempDir()
	annotations := filepath.Join(dir, "annotations.json")
	require.NoError(t, os.WriteFile(annotations, []byte(`{"version":1,"coordinateSpace":{"width":4,"height":3},"annotations":[{"id":"changed-area","label":"Changed area","bounds":{"x":0,"y":0,"width":4,"height":3}}]}`), 0o600))

	run := func(name string, args ...string) diff.ImageComparison {
		output, err := pixelPerfectCommand(binary, append([]string{
			filepath.Join(fixtures, "reference.png"),
			filepath.Join(fixtures, "two-regions.png"),
			"--output", filepath.Join(dir, name+".png"),
		}, args...)...).CombinedOutput()
		require.NoError(t, err, string(output))
		var result diff.ImageComparison
		require.NoError(t, json.Unmarshal(output, &result))
		return result
	}

	plain := run("plain")
	enriched := run("enriched", "--annotations", annotations)

	assert.Equal(t, plain.ChangedPixels, enriched.ChangedPixels)
	assert.Equal(t, plain.RMSE, enriched.RMSE)
	require.NotEmpty(t, enriched.Regions)
	for _, region := range enriched.Regions {
		require.Len(t, region.Annotations, 1)
		assert.Equal(t, "changed-area", region.Annotations[0].ID)
	}
}

func TestPixelPerfectComparisonProfileAndExplicitOverride(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	dir := t.TempDir()
	profile := filepath.Join(dir, "pxp.json")
	require.NoError(t, os.WriteFile(profile, []byte(`{"version":1,"suggestOffset":2,"regionGap":3,"minRegionPixels":2}`), 0o600))
	mask := filepath.Join(dir, "mask.png")

	output, err := pixelPerfectCommand(binary,
		filepath.Join(fixtures, "offset-reference.png"),
		filepath.Join(fixtures, "offset-right-one.png"),
		"--profile", profile,
		"--suggest-offset", "0",
		"--output", mask,
	).CombinedOutput()

	require.NoError(t, err, string(output))
	type resolvedInt struct {
		Value  int    `json:"value"`
		Source string `json:"source"`
	}
	var result struct {
		SuggestedOffset *diff.SuggestedOffset `json:"suggestedOffset"`
		Configuration   struct {
			Resolved struct {
				SuggestOffset   resolvedInt `json:"suggestOffset"`
				RegionGap       resolvedInt `json:"regionGap"`
				MinRegionPixels resolvedInt `json:"minRegionPixels"`
			} `json:"resolved"`
		} `json:"configuration"`
	}
	require.NoError(t, json.Unmarshal(output, &result))
	assert.Nil(t, result.SuggestedOffset)
	assert.Equal(t, 0, result.Configuration.Resolved.SuggestOffset.Value)
	assert.Equal(t, "flag", result.Configuration.Resolved.SuggestOffset.Source)
	assert.Equal(t, 3, result.Configuration.Resolved.RegionGap.Value)
	assert.Equal(t, "profile", result.Configuration.Resolved.RegionGap.Source)
	assert.Equal(t, 2, result.Configuration.Resolved.MinRegionPixels.Value)
	assert.Equal(t, "profile", result.Configuration.Resolved.MinRegionPixels.Source)
	assertPNGDimensions(t, mask, 4, 3)
}

func TestPixelPerfectOffsetInterpretationRejectsUpstreamNegativeControls(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "upstream", "odiff")
	tests := []struct {
		name, reference, actual string
	}{
		{name: "antialiasing change", reference: "antialiasing-on.png", actual: "antialiasing-off.png"},
		{name: "equivalent extreme alpha", reference: "extreme-alpha.png", actual: "extreme-alpha-1.png"},
		{name: "color and content change", reference: "orange.png", actual: "orange_changed.png"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mask := filepath.Join(t.TempDir(), "mask.png")
			output, err := pixelPerfectCommand(binary,
				filepath.Join(fixtures, test.reference),
				filepath.Join(fixtures, test.actual),
				"--suggest-offset", "3",
				"--suggest-movement", "3",
				"--output", mask,
			).CombinedOutput()
			require.NoError(t, err, string(output))
			var comparison diff.ImageComparison
			require.NoError(t, json.Unmarshal(output, &comparison))
			require.NotNil(t, comparison.SuggestedOffset)
			assert.Equal(t, "inconclusive", comparison.SuggestedOffset.Interpretation)
			assert.Empty(t, comparison.MovedRegions)
		})
	}
}

func TestPixelPerfectAppliesExplicitCropFixtures(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	mask := filepath.Join(t.TempDir(), "mask.png")
	output, err := pixelPerfectCommand(binary,
		filepath.Join(fixtures, "crop-reference.png"), filepath.Join(fixtures, "crop-actual.png"),
		"--reference-crop", "1,0,2,2",
		"--actual-crop", "0,0,2,2",
		"--output", mask,
	).CombinedOutput()

	require.NoError(t, err, string(output))
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal(output, &comparison))
	assert.Equal(t, 2, comparison.Width)
	assert.Equal(t, 2, comparison.Height)
	assert.Equal(t, 0, comparison.ChangedPixels)
	assert.Empty(t, comparison.Regions)
	require.NotNil(t, comparison.Inputs)
	assert.Equal(t, diff.ImageInput{Width: 4, Height: 2, Crop: &diff.Bounds{X: 1, Y: 0, Width: 2, Height: 2}}, comparison.Inputs.Reference)
	assert.Equal(t, diff.ImageInput{Width: 2, Height: 2, Crop: &diff.Bounds{X: 0, Y: 0, Width: 2, Height: 2}}, comparison.Inputs.Actual)
	assertPNGDimensions(t, mask, 2, 2)
}

func TestPixelPerfectExplicitCropReportsOriginalInputBounds(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	mask := filepath.Join(t.TempDir(), "mask.png")
	output, err := pixelPerfectCommand(binary,
		filepath.Join(fixtures, "crop-reference.png"), filepath.Join(fixtures, "crop-actual-changed.png"),
		"--reference-crop", "1,0,2,2",
		"--actual-crop", "0,0,2,2",
		"--output", mask,
	).CombinedOutput()

	require.NoError(t, err, string(output))
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal(output, &comparison))
	require.Len(t, comparison.Regions, 1)
	assert.Equal(t, diff.Bounds{X: 1, Y: 1, Width: 1, Height: 1}, comparison.Regions[0].Bounds)
	assert.Equal(t, &diff.InputBounds{
		Reference: diff.Bounds{X: 2, Y: 1, Width: 1, Height: 1},
		Actual:    diff.Bounds{X: 1, Y: 1, Width: 1, Height: 1},
	}, comparison.Regions[0].InputBounds)
}

func TestPixelPerfectExplicitCropFixtureErrors(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	reference := filepath.Join(fixtures, "crop-reference.png")
	actual := filepath.Join(fixtures, "crop-actual.png")
	tests := []struct {
		name, expected string
		flags          []string
	}{
		{name: "out of bounds", flags: []string{"--actual-crop", "1,0,2,2"}, expected: "invalid --actual-crop: crop 1,0,2,2 is outside image bounds 2x2"},
		{name: "dimension mismatch", flags: []string{"--reference-crop", "1,0,2,2", "--actual-crop", "0,0,1,2"}, expected: "cropped image dimensions differ: reference is 2x2, actual is 1x2"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := []string{reference, actual, "--output", filepath.Join(t.TempDir(), "mask.png")}
			args = append(args, test.flags...)
			output, err := pixelPerfectCommand(binary, args...).CombinedOutput()

			require.Error(t, err)
			assert.Contains(t, string(output), test.expected)
		})
	}
}

func TestPixelPerfectOmitsOffsetWhenMaskExcludesAllPixels(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	mask := filepath.Join(t.TempDir(), "mask.png")
	output, err := pixelPerfectCommand(binary,
		filepath.Join(fixtures, "reference.png"), filepath.Join(fixtures, "two-regions.png"),
		"--output", mask,
		"--mask", filepath.Join(fixtures, "all-excluded-mask.png"),
		"--suggest-offset", "2",
	).CombinedOutput()

	require.NoError(t, err, string(output))
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal(output, &comparison))
	assert.Zero(t, comparison.ComparedPixels)
	assert.Nil(t, comparison.SuggestedOffset)
}

func TestPixelPerfectRejectsWrongSizeComparisonMask(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	mask := filepath.Join(t.TempDir(), "mask.png")
	output, err := pixelPerfectCommand(binary,
		filepath.Join(fixtures, "reference.png"), filepath.Join(fixtures, "two-regions.png"),
		"--output", mask, "--mask", filepath.Join(fixtures, "wrong-size-mask.png"),
	).CombinedOutput()

	require.Error(t, err)
	assert.Contains(t, string(output), `"message": "Command could not complete."`)
	_, statErr := os.Stat(mask)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestPixelPerfectNoArgsShowsCompactNextSteps(t *testing.T) {
	binary := buildCommand(t, "pxp")

	output, err := exec.Command(binary).CombinedOutput()

	require.NoError(t, err, string(output))
	assert.Contains(t, string(output), "pxp compares PNG screenshots")
	resolvedBinary, resolveErr := filepath.EvalSymlinks(binary)
	require.NoError(t, resolveErr)
	assert.Contains(t, string(output), "Executable: "+resolvedBinary)
	assert.Contains(t, string(output), "pxp reference.png actual.png")
	assert.Less(t, len(output), 400)
}

func TestPixelPerfectCLIErrorContracts(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	reference := filepath.Join(fixtures, "reference.png")
	actual := filepath.Join(fixtures, "two-regions.png")
	tests := []struct {
		name, expected string
		args           func(*testing.T) []string
	}{
		{name: "missing argument", expected: `"category": "usage"`, args: func(*testing.T) []string { return []string{"reference.png"} }},
		{name: "unknown flag", expected: `"message": "unknown flag: --unknown"`, args: func(*testing.T) []string { return []string{reference, actual, "--unknown"} }},
		{name: "malformed PNG", expected: `"message": "Could not decode an input image."`, args: func(t *testing.T) []string {
			invalid := filepath.Join(t.TempDir(), "invalid.png")
			require.NoError(t, os.WriteFile(invalid, []byte("not a png"), 0o600))
			return []string{reference, invalid, "--output", filepath.Join(t.TempDir(), "mask.png")}
		}},
		{name: "output is directory", expected: `"message": "Could not access a required file."`, args: func(t *testing.T) []string {
			return []string{reference, actual, "--output", t.TempDir()}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := pixelPerfectCommand(binary, test.args(t)...).CombinedOutput()
			require.Error(t, err)
			assert.Contains(t, string(output), test.expected)
		})
	}
}

func TestPixelPerfectMissingFileErrorRedactsAbsolutePath(t *testing.T) {
	binary := buildCommand(t, "pxp")
	missing := filepath.Join(t.TempDir(), "private-reference.png")
	output, err := pixelPerfectCommand(binary, missing, "actual.png").CombinedOutput()

	require.Error(t, err)
	assert.Contains(t, string(output), `"message": "Could not access a required file."`)
	assert.NotContains(t, string(output), missing)
}

func TestPixelPerfectUnknownFlagRetainsCorrectionAndRecovery(t *testing.T) {
	binary := buildCommand(t, "pxp")
	output, err := pixelPerfectCommand(binary, "reference.png", "actual.png", "--threshol", "8").CombinedOutput()

	require.Error(t, err)
	assert.Contains(t, string(output), "Did you mean `--threshold`?")
	assert.Contains(t, string(output), "Run `pxp --help` for valid flags.")
	assert.Contains(t, string(output), `"exitCode": 2`)
}

func TestPixelPerfectHelpDocumentsStandaloneContract(t *testing.T) {
	binary := buildCommand(t, "pxp")
	output, err := pixelPerfectCommand(binary, "--help").CombinedOutput()

	require.NoError(t, err, string(output))
	help := string(output)
	assert.Contains(t, help, "pxp <reference.png> <actual.png>")
	for _, flag := range []string{"--output", "--overlay", "--region", "--reference-crop", "--actual-crop", "--ignore-region", "--mask", "--threshold", "--perceptual-threshold", "--suggest-offset", "--suggest-movement", "--max-rmse", "--max-changed-ratio", "--max-perceptual-changed-ratio"} {
		assert.Contains(t, help, flag)
	}
}

func TestPixelPerfectRejectsRegionsOutsideImage(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	reference := filepath.Join(fixtures, "reference.png")
	actual := filepath.Join(fixtures, "two-regions.png")
	for _, region := range []string{"0,0,0,0", "-1,0,2,2", "100,100,2,2", "3,2,5,5"} {
		t.Run(region, func(t *testing.T) {
			mask := filepath.Join(t.TempDir(), "mask.png")
			output, err := pixelPerfectCommand(binary, reference, actual, "--output", mask, "--region", region).CombinedOutput()

			require.Error(t, err)
			assert.Contains(t, string(output), `"message": "Command could not complete."`)
			_, statErr := os.Stat(mask)
			assert.ErrorIs(t, statErr, os.ErrNotExist)
		})
	}
}

func TestPixelPerfectRejectsArtifactPathCollisions(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	reference := filepath.Join(fixtures, "reference.png")
	actual := filepath.Join(fixtures, "two-regions.png")
	tests := []struct {
		name, expected string
		args           func(*testing.T) []string
	}{
		{name: "mask overwrites reference", expected: "--output must not overwrite an input image", args: func(*testing.T) []string { return []string{reference, actual, "--output", reference} }},
		{name: "mask overwrites actual", expected: "--output must not overwrite an input image", args: func(*testing.T) []string { return []string{reference, actual, "--output", actual} }},
		{name: "overlay overwrites mask", expected: "--overlay must differ from --output", args: func(t *testing.T) []string {
			artifact := filepath.Join(t.TempDir(), "artifact.png")
			return []string{reference, actual, "--output", artifact, "--overlay", artifact}
		}},
		{name: "overlay overwrites reference", expected: "--overlay must not overwrite an input image", args: func(t *testing.T) []string {
			return []string{reference, actual, "--output", filepath.Join(t.TempDir(), "mask.png"), "--overlay", reference}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := pixelPerfectCommand(binary, test.args(t)...).CombinedOutput()
			require.Error(t, err)
			assert.Contains(t, string(output), test.expected)
		})
	}
}

func TestPixelPerfectRejectsInvalidFlagValues(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	reference := filepath.Join(fixtures, "reference.png")
	actual := filepath.Join(fixtures, "two-regions.png")
	tests := []struct {
		name, expected string
		flags          []string
	}{
		{name: "threshold overflow", flags: []string{"--threshold", "256"}, expected: "value out of range"},
		{name: "malformed region", flags: []string{"--region", "0,0,4"}, expected: "--region must be x,y,width,height"},
		{name: "non numeric region", flags: []string{"--region", "0,zero,4,3"}, expected: "--region must contain integers"},
		{name: "malformed ignored region", flags: []string{"--ignore-region", "0,0,4"}, expected: "invalid --ignore-region"},
		{name: "empty ignored region", flags: []string{"--ignore-region", "0,0,0,1"}, expected: "invalid --ignore-region: width and height must be positive"},
		{name: "negative ignored region size", flags: []string{"--ignore-region", "0,0,1,-1"}, expected: "invalid --ignore-region: width and height must be positive"},
		{name: "negative offset radius", flags: []string{"--suggest-offset", "-1"}, expected: "--suggest-offset must be non-negative"},
		{name: "negative movement radius", flags: []string{"--suggest-movement", "-1"}, expected: "--suggest-movement must be non-negative"},
		{name: "negative region gap", flags: []string{"--region-gap", "-1"}, expected: "--region-gap must be non-negative"},
		{name: "zero minimum region pixels", flags: []string{"--min-region-pixels", "0"}, expected: "--min-region-pixels must be positive"},
		{name: "negative perceptual threshold", flags: []string{"--perceptual-threshold", "-0.1"}, expected: "--perceptual-threshold must be a finite non-negative number"},
		{name: "NaN perceptual threshold", flags: []string{"--perceptual-threshold", "NaN"}, expected: "--perceptual-threshold must be a finite non-negative number"},
		{name: "NaN RMSE limit", flags: []string{"--max-rmse", "NaN"}, expected: "--max-rmse must be -1 or a finite non-negative number"},
		{name: "invalid negative RMSE limit", flags: []string{"--max-rmse", "-2"}, expected: "--max-rmse must be -1 or a finite non-negative number"},
		{name: "changed ratio above one", flags: []string{"--max-changed-ratio", "1.1"}, expected: "--max-changed-ratio must be -1 or between 0 and 1"},
		{name: "invalid negative changed ratio", flags: []string{"--max-changed-ratio", "-2"}, expected: "--max-changed-ratio must be -1 or between 0 and 1"},
		{name: "perceptual changed ratio above one", flags: []string{"--max-perceptual-changed-ratio", "1.1"}, expected: "--max-perceptual-changed-ratio must be -1 or between 0 and 1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := []string{reference, actual, "--output", filepath.Join(t.TempDir(), "mask.png")}
			args = append(args, test.flags...)
			output, err := pixelPerfectCommand(binary, args...).CombinedOutput()
			require.Error(t, err)
			assert.Contains(t, string(output), test.expected)
		})
	}
}

func TestPixelPerfectCombinesRegionAndRepeatedIgnores(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	mask := filepath.Join(t.TempDir(), "mask.png")
	output, err := pixelPerfectCommand(binary,
		filepath.Join(fixtures, "reference.png"), filepath.Join(fixtures, "two-regions.png"),
		"--output", mask,
		"--region", "0,0,4,2",
		"--ignore-region", "0,0,1,1",
		"--ignore-region", "1,0,1,1",
	).CombinedOutput()

	require.NoError(t, err, string(output))
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal(output, &comparison))
	assert.Equal(t, 0, comparison.ChangedPixels)
	assert.Equal(t, 6, comparison.ComparedPixels)
	assert.Equal(t, &diff.Bounds{X: 0, Y: 0, Width: 4, Height: 2}, comparison.ComparedRegion)
	assertPNGDimensions(t, mask, 4, 2)
}

func TestPixelPerfectOutputIsDeterministic(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	mask := filepath.Join(t.TempDir(), "mask.png")
	args := []string{
		filepath.Join(fixtures, "real-ui-reference.png"),
		filepath.Join(fixtures, "real-ui-implementation.png"),
		"--output", mask, "--region-gap", "8", "--min-region-pixels", "12",
	}

	first, err := pixelPerfectCommand(binary, args...).CombinedOutput()
	require.NoError(t, err, string(first))
	firstMask, err := os.ReadFile(mask)
	require.NoError(t, err)
	second, err := pixelPerfectCommand(binary, args...).CombinedOutput()
	require.NoError(t, err, string(second))
	secondMask, err := os.ReadFile(mask)
	require.NoError(t, err)

	assert.Equal(t, string(first), string(second))
	assert.Equal(t, firstMask, secondMask)
}

func TestPixelPerfectRealUIScreenshot(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	mask := filepath.Join(t.TempDir(), "mask.png")
	overlay := filepath.Join(t.TempDir(), "overlay.png")
	output, err := pixelPerfectCommand(binary,
		filepath.Join(fixtures, "real-ui-reference.png"),
		filepath.Join(fixtures, "real-ui-implementation.png"),
		"--output", mask,
		"--overlay", overlay,
		"--region-gap", "8",
		"--min-region-pixels", "12",
	).CombinedOutput()

	require.NoError(t, err, string(output))
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal(output, &comparison))
	assert.Equal(t, 575, comparison.Width)
	assert.Equal(t, 477, comparison.Height)
	assert.Equal(t, 575*477, comparison.ComparedPixels)
	assert.Greater(t, comparison.ChangedPixels, 1_000)
	assert.Greater(t, comparison.RMSE, 0.0)
	assert.Greater(t, comparison.EdgeRMSE, 0.0)
	assert.NotEmpty(t, comparison.Regions)
	assert.LessOrEqual(t, len(comparison.Regions), 20)
	assertPNGDimensions(t, mask, 575, 477)
	assertPNGDimensions(t, overlay, 575, 477)
}

func TestPixelPerfectAcceptsPerceptualThresholdAboveBlackWhiteDistance(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	mask := filepath.Join(t.TempDir(), "mask.png")
	output, err := pixelPerfectCommand(binary,
		filepath.Join(fixtures, "reference.png"), filepath.Join(fixtures, "two-regions.png"),
		"--output", mask, "--perceptual-threshold", "1.1",
	).CombinedOutput()

	require.NoError(t, err, string(output))
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal(output, &comparison))
	assert.Equal(t, 1.1, comparison.PerceptualThreshold)
	assert.Zero(t, comparison.PerceptualChangedPixels)
	assert.Equal(t, 2, comparison.ChangedPixels)
}

func TestPixelPerfectValidationGateBoundaries(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	reference := filepath.Join(fixtures, "reference.png")
	actual := filepath.Join(fixtures, "two-regions.png")
	baselineMask := filepath.Join(t.TempDir(), "baseline.png")
	baselineOutput, err := pixelPerfectCommand(binary, reference, actual, "--output", baselineMask).CombinedOutput()
	require.NoError(t, err, string(baselineOutput))
	var baseline diff.ImageComparison
	require.NoError(t, json.Unmarshal(baselineOutput, &baseline))

	tests := []struct {
		name, flag string
		limit      float64
		passes     bool
		expected   string
	}{
		{name: "RMSE accepts exact boundary", flag: "--max-rmse", limit: baseline.RMSE, passes: true},
		{name: "RMSE rejects below boundary", flag: "--max-rmse", limit: baseline.RMSE / 2, expected: "RMSE"},
		{name: "changed ratio accepts exact boundary", flag: "--max-changed-ratio", limit: baseline.ChangedRatio, passes: true},
		{name: "changed ratio rejects below boundary", flag: "--max-changed-ratio", limit: baseline.ChangedRatio / 2, expected: "changed ratio"},
		{name: "perceptual ratio accepts exact boundary", flag: "--max-perceptual-changed-ratio", limit: baseline.PerceptualChangedRatio, passes: true},
		{name: "perceptual ratio rejects below boundary", flag: "--max-perceptual-changed-ratio", limit: baseline.PerceptualChangedRatio / 2, expected: "perceptual changed ratio"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mask := filepath.Join(t.TempDir(), "mask.png")
			command := pixelPerfectCommand(binary, reference, actual, "--output", mask, test.flag, fmt.Sprintf("%.17g", test.limit))
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			err := command.Run()
			if test.passes {
				require.NoError(t, err, stderr.String())
				var comparison diff.ImageComparison
				require.NoError(t, json.Unmarshal(stdout.Bytes(), &comparison))
				return
			}
			require.Error(t, err)
			assert.Contains(t, stderr.String(), "image diff validation failed")
			assert.Contains(t, stderr.String(), test.expected)
			var evidence struct {
				Validation struct {
					Passed bool  `json:"passed"`
					Failed []any `json:"failed"`
				} `json:"validation"`
			}
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &evidence))
			assert.False(t, evidence.Validation.Passed)
			assert.NotEmpty(t, evidence.Validation.Failed)
			assertPNGDimensions(t, mask, 4, 3)
		})
	}
}

func TestPixelPerfectRealUIValidationGate(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	mask := filepath.Join(t.TempDir(), "mask.png")
	output, err := pixelPerfectCommand(binary,
		filepath.Join(fixtures, "real-ui-reference.png"),
		filepath.Join(fixtures, "real-ui-implementation.png"),
		"--output", mask,
		"--max-changed-ratio", "0.001",
	).CombinedOutput()

	require.Error(t, err)
	assert.Contains(t, string(output), "image diff validation failed: changed ratio")
	assertPNGDimensions(t, mask, 575, 477)
}

func TestPixelPerfectRejectsRealUIWithUnequalDimensions(t *testing.T) {
	binary := buildCommand(t, "pxp")
	fixtures := filepath.Join("fixtures", "image-diff")
	mask := filepath.Join(t.TempDir(), "mask.png")
	output, err := pixelPerfectCommand(binary,
		filepath.Join(fixtures, "real-ui-reference.png"),
		filepath.Join(fixtures, "unequal-dimensions.png"),
		"--output", mask,
	).CombinedOutput()

	require.Error(t, err)
	assert.Contains(t, string(output), `"message": "Command could not complete."`)
	_, statErr := os.Stat(mask)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func copyFile(source, destination string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0o600)
}

func assertPNGDimensions(t *testing.T, path string, width, height int) {
	t.Helper()
	file, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	config, err := png.DecodeConfig(file)
	require.NoError(t, err)
	assert.Equal(t, width, config.Width)
	assert.Equal(t, height, config.Height)
}

func pixelPerfectCommand(binary string, args ...string) *exec.Cmd {
	return exec.Command(binary, append([]string{"--json"}, args...)...)
}

func buildCommand(t *testing.T, name string) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), name)
	command := exec.Command("go", "build", "-o", binary, "./cmd/"+name)
	command.Dir = filepath.Join("..", "..")
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	return binary
}
