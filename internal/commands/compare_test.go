package commands

import (
	"bytes"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"

	clipkg "github.com/cristianoliveira/pxp/internal/cli"
	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIgnoredRegionsRejectsMalformedAndEmptyRegions(t *testing.T) {
	ignored, err := parseIgnoredRegions([]string{"1,2,3,4", "5,6,7,8"})
	require.NoError(t, err)
	assert.Equal(t, []diff.Bounds{{X: 1, Y: 2, Width: 3, Height: 4}, {X: 5, Y: 6, Width: 7, Height: 8}}, ignored)
	assert.EqualError(t, parseIgnoredRegionsError([]string{"invalid"}), "invalid --ignore-region: --region must be x,y,width,height")
	assert.EqualError(t, parseIgnoredRegionsError([]string{"1,2,0,4"}), "invalid --ignore-region: width and height must be positive")
}

func TestComparisonRegionControlValidationRejectsInvalidValues(t *testing.T) {
	assert.EqualError(t, validateRegionControls(-1, 0, 0, 1, 1), "--suggest-offset must be non-negative")
	assert.EqualError(t, validateRegionControls(0, -1, 0, 1, 1), "--suggest-movement must be non-negative")
	assert.EqualError(t, validateRegionControls(0, 0, -1, 1, 1), "--region-gap must be non-negative")
	assert.EqualError(t, validateRegionControls(0, 0, 0, 0, 1), "--min-region-pixels must be positive")
	assert.EqualError(t, validateRegionControls(0, 0, 0, 1, 0), "--max-regions must be positive")
	assert.NoError(t, validateRegionControls(1, 1, 0, 1, 20))
}

func TestComparisonThresholdValidationRejectsInvalidValues(t *testing.T) {
	assert.EqualError(t, validateComparisonThresholds(-1, -1, -1, -1), "--perceptual-threshold must be a finite non-negative number")
	assert.EqualError(t, validateComparisonThresholds(1, -2, -1, -1), "--max-rmse must be -1 or a finite non-negative number")
	assert.EqualError(t, validateComparisonThresholds(1, -1, 2, -1), "--max-changed-ratio must be -1 or between 0 and 1")
	assert.EqualError(t, validateComparisonThresholds(1, -1, -1, 2), "--max-perceptual-changed-ratio must be -1 or between 0 and 1")
	assert.NoError(t, validateComparisonThresholds(0, -1, 0.5, 1))
}

func TestComparisonRequestValidationRejectsUnsafeArtifactPathsAndUnknownProvider(t *testing.T) {
	assert.EqualError(t, validateComparisonArtifactPaths("reference.png", "actual.png", "reference.png", "", ""), "--output must not overwrite an input image")
	assert.EqualError(t, validateComparisonArtifactPaths("reference.png", "actual.png", "mask.png", "mask.png", ""), "--overlay must differ from --output")
	assert.EqualError(t, validateComparisonArtifactPaths("reference.png", "actual.png", "mask.png", "overlay.png", "mask.png"), "--report must not overwrite an input, mask, or overlay")
	assert.EqualError(t, validateVisualContextProvider(true, "unknown"), `unsupported visual context provider "unknown"`)
	assert.NoError(t, validateComparisonArtifactPaths("reference.png", "actual.png", "mask.png", "overlay.png", "report.html"))
	assert.NoError(t, validateVisualContextProvider(true, "openrouter"))
}

func TestComparisonDefaultsToTOON(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	command := NewCommand()
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetArgs([]string{reference, actual, "--output", filepath.Join(t.TempDir(), "diff.png")})

	require.NoError(t, command.Execute())
	assert.Contains(t, stdout.String(), "changedPixels:")
	assert.NotContains(t, stdout.String(), `"changedPixels"`)
}

func TestCommandNoArgsShowsCompactNextSteps(t *testing.T) {
	result := executeCommand(newCommandWithExecutable(func() (string, error) { return "~/bin/pxp", nil }))

	require.NoError(t, result.Err)
	assert.Contains(t, result.Stdout, "pxp harnesses image comparison")
	assert.Contains(t, result.Stdout, "Executable: ~/bin/pxp")
	assert.Contains(t, result.Stdout, "pxp reference.png actual.png")
	assert.Contains(t, result.Stdout, "pxp probe --help")
	assert.Contains(t, result.Stdout, "pxp scan --help")
	assert.Less(t, len(result.Stdout), 400)
}

func TestCommandUsageErrorsShowCorrectionsAndLocalExamples(t *testing.T) {
	missing := executeCommand(NewCommand(), "reference.png")
	require.Error(t, missing.Err)
	assert.ErrorContains(t, missing.Err, "requires <reference.png> and <actual.png>")
	assert.ErrorContains(t, missing.Err, "pxp reference.png actual.png")

	unknown := executeCommand(NewCommand(), "reference.png", "actual.png", "--threshol", "8")
	require.Error(t, unknown.Err)
	assert.ErrorContains(t, unknown.Err, "unknown flag: --threshol")
	assert.ErrorContains(t, unknown.Err, "Did you mean `--threshold`?")
	assert.ErrorContains(t, unknown.Err, "Run `pxp --help` for valid flags.")
	assert.NotContains(t, unknown.Err.Error(), "Available flags")
	assert.Less(t, len(unknown.Err.Error()), 180)

	help := executeCommand(NewCommand(), "probe", "--help")
	require.NoError(t, help.Err)
	assert.Contains(t, help.Stdout, "Examples:")
	assert.Contains(t, help.Stdout, "pxp probe reference.png actual.png --at 12,24")
}

func TestDiffImageCommandProducesMaskAndJSONMetrics(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 2, 2)))

	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--output", mask)

	require.NoError(t, result.Err)
	assert.JSONEq(t, `{"width":2,"height":2,"changedPixels":0,"comparedPixels":4,"changedRatio":0,"rmse":0,"rgbRmse":0,"luminanceRmse":0,"alphaRmse":0,"edgeRmse":0,"perceptualRmse":0,"perceptualChangedPixels":0,"perceptualChangedRatio":0,"perceptualThreshold":0.1,"antialiasedPixels":0,"evidence":{"rawOnlyPixels":0,"perceptualOnlyPixels":0,"rawAndPerceptualPixels":0},"mask":"`+mask+`"}`, result.Stdout)
}

func TestDiffImageCommandWritesDefaultMaskOutput(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	defaultMask := filepath.Join(dir, "actual.diff.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 2, 2)))

	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--threshold", "8")

	require.NoError(t, result.Err)
	assert.JSONEq(t, `{"width":2,"height":2,"changedPixels":0,"comparedPixels":4,"changedRatio":0,"rmse":0,"rgbRmse":0,"luminanceRmse":0,"alphaRmse":0,"edgeRmse":0,"perceptualRmse":0,"perceptualChangedPixels":0,"perceptualChangedRatio":0,"perceptualThreshold":0.1,"antialiasedPixels":0,"evidence":{"rawOnlyPixels":0,"perceptualOnlyPixels":0,"rawAndPerceptualPixels":0},"mask":"`+defaultMask+`"}`, result.Stdout)
	_, err := os.Stat(defaultMask)
	require.NoError(t, err)
}

func TestDiffImageCommandWritesReportWithoutExplicitMaskOutput(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	report := filepath.Join(dir, "report.html")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 2, 2)))

	defaultMask := filepath.Join(dir, "actual.diff.png")
	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--report", report)

	require.NoError(t, result.Err)
	assert.Contains(t, result.Stdout, defaultMask)
	_, statErr := os.Stat(defaultMask)
	require.NoError(t, statErr)
	content, err := os.ReadFile(report)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Pixel Perfect Report")
	assert.Contains(t, string(content), "data:image/png;base64,")
}

func TestDiffImageCommandPrefersMetadataLogicalCrop(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	metadata := filepath.Join(dir, "reference.export.json")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 4, 4)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	require.NoError(t, os.WriteFile(metadata, []byte(`{"version":1,"nodeBounds":{"width":2,"height":2},"exportBounds":{"width":4,"height":4},"logicalCrop":{"x":1,"y":0,"width":2,"height":2}}`), 0o600))

	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--reference-metadata", metadata, "--output", filepath.Join(dir, "mask.png"))

	require.NoError(t, result.Err)
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &comparison))
	require.NotNil(t, comparison.Inputs)
	assert.Equal(t, &diff.Bounds{X: 1, Y: 0, Width: 2, Height: 2}, comparison.Inputs.Reference.Crop)
}

func TestDiffImageCommandWritesHTMLReport(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	report := filepath.Join(dir, "report.html")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 2, 2)))

	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--output", mask, "--report", report)

	require.NoError(t, result.Err)
	content, err := os.ReadFile(report)
	require.NoError(t, err)
	html := string(content)
	assert.Contains(t, html, "Pixel Perfect Report")
	assert.Contains(t, html, "Global metrics")
	assert.Contains(t, html, "data:image/png;base64,")
	assert.Contains(t, html, "Reference")
	assert.Contains(t, html, "Actual")
	assert.Contains(t, html, "Mask")
	assert.Contains(t, html, "Threshold: 0")
	assert.Contains(t, html, "Perceptual threshold: 0.1")
}

func TestDiffImageCommandHTMLReportIncludesCropAndRegionProvenance(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	report := filepath.Join(dir, "report.html")
	referenceImage := image.NewRGBA(image.Rect(0, 0, 4, 2))
	actualImage := image.NewRGBA(image.Rect(0, 0, 2, 2))
	actualImage.Set(1, 1, image.White)
	writeTestPNG(t, reference, referenceImage)
	writeTestPNG(t, actual, actualImage)

	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--reference-crop", "1,0,2,2", "--actual-crop", "0,0,2,2", "--region", "0,0,2,2", "--output", mask, "--report", report)

	require.NoError(t, result.Err)
	content, err := os.ReadFile(report)
	require.NoError(t, err)
	html := string(content)
	assert.Contains(t, html, "Reference crop: 1,0,2,2")
	assert.Contains(t, html, "Actual crop: 0,0,2,2")
	assert.Contains(t, html, "Compared region: 0,0,2,2")
	assert.Contains(t, html, "1,1,1,1")
	assert.Contains(t, html, "2,1,1,1")
}

func TestDiffImageCommandHelpDocumentsVisualContextPrompt(t *testing.T) {
	command := newCommand(diff.CompareImagesWithThresholds)
	result := executeCommand(command, "--help")

	require.NoError(t, result.Err)
	assert.Contains(t, result.Stdout, "--visual-context-prompt")
}

func TestDiffImageCommandRejectsReportPathCollisions(t *testing.T) {
	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), "reference.png", "actual.png", "--output", "mask.png", "--report", "mask.png")

	assert.EqualError(t, result.Err, "--report must not overwrite an input, mask, or overlay")
}

func TestDiffImageCommandValidatesVisualContextProviderBeforeReadingImages(t *testing.T) {
	result := executeCommand(NewCommand(), "missing-reference.png", "missing-actual.png", "--visual-context", "--visual-context-provider", "unsupported")

	require.Error(t, result.Err)
	assert.ErrorContains(t, result.Err, `unsupported visual context provider "unsupported"`)
	assert.Equal(t, 2, clipkg.ExitCode(result.Err))
	assert.NotContains(t, result.Err.Error(), "decode reference")
}

func TestDiffImageCommandAddsDisclaimerWhenVisualContextIsNotConfigured(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PI_SPECTACLES_CONFIG", filepath.Join(dir, "missing.json"))
	t.Setenv("OPENROUTER_API_KEY", "")
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 2, 2)))

	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--output", mask, "--visual-context")

	require.NoError(t, result.Err)
	assert.JSONEq(t, `{"width":2,"height":2,"changedPixels":0,"comparedPixels":4,"changedRatio":0,"rmse":0,"rgbRmse":0,"luminanceRmse":0,"alphaRmse":0,"edgeRmse":0,"perceptualRmse":0,"perceptualChangedPixels":0,"perceptualChangedRatio":0,"perceptualThreshold":0.1,"antialiasedPixels":0,"evidence":{"rawOnlyPixels":0,"perceptualOnlyPixels":0,"rawAndPerceptualPixels":0},"mask":"`+mask+`","visualContext":{"provider":"openrouter","advisory":true,"disclaimer":"Visual context unavailable: configure openrouter credentials in the Pi Spectacles config or environment."}}`, result.Stdout)
}

func TestLimitImageRegionsReportsTruncationAndSupportsFullOutput(t *testing.T) {
	regions := make([]diff.Region, 25)

	limited, count, truncated := limitImageRegions(regions, 20, false)
	assert.Len(t, limited, 20)
	assert.Equal(t, 25, count)
	assert.True(t, truncated)
	encoded, err := json.Marshal(outputEnvelope{RegionCount: count, RegionsTruncated: truncated})
	require.NoError(t, err)
	assert.JSONEq(t, `{"regionCount":25,"regionsTruncated":true,"width":0,"height":0,"changedPixels":0,"comparedPixels":0,"changedRatio":0,"rmse":0,"rgbRmse":0,"luminanceRmse":0,"alphaRmse":0,"edgeRmse":0,"perceptualRmse":0,"perceptualChangedPixels":0,"perceptualChangedRatio":0,"perceptualThreshold":0,"antialiasedPixels":0,"evidence":{"rawOnlyPixels":0,"perceptualOnlyPixels":0,"rawAndPerceptualPixels":0}}`, string(encoded))

	full, count, truncated := limitImageRegions(regions, 20, true)
	assert.Len(t, full, 25)
	assert.Equal(t, 25, count)
	assert.False(t, truncated)
}

func TestDiffImageCommandClassifiesInvalidOptionsAsUsageErrorsBeforeReadingImages(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		error string
	}{
		{name: "region limit", args: []string{"--max-regions", "0"}, error: "--max-regions must be positive"},
		{name: "changed ratio", args: []string{"--max-changed-ratio", "2"}, error: "--max-changed-ratio must be -1 or between 0 and 1"},
		{name: "output collision", args: []string{"--output", "missing-reference.png"}, error: "--output must not overwrite an input image"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"missing-reference.png", "missing-actual.png"}, test.args...)
			result := executeCommand(NewCommand(), args...)

			require.Error(t, result.Err)
			assert.ErrorContains(t, result.Err, test.error)
			assert.Equal(t, 2, clipkg.ExitCode(result.Err))
			assert.NotContains(t, result.Err.Error(), "decode reference")
		})
	}
}

func TestDiffImageCommandClassifiesMissingInputAsOperationalError(t *testing.T) {
	result := executeCommand(NewCommand(), "missing-reference.png", "missing-actual.png")

	require.Error(t, result.Err)
	assert.Equal(t, 1, clipkg.ExitCode(result.Err))
}

func TestGroupImageRegionsMergesNearbyClusters(t *testing.T) {
	regions := []diff.Region{
		{Bounds: diff.Bounds{X: 0, Y: 0, Width: 2, Height: 2}, ChangedPixels: 3},
		{Bounds: diff.Bounds{X: 4, Y: 1, Width: 2, Height: 2}, ChangedPixels: 4},
		{Bounds: diff.Bounds{X: 20, Y: 20, Width: 1, Height: 1}, ChangedPixels: 1},
	}

	grouped := groupImageRegions(regions, 2)

	require.Len(t, grouped, 2)
	assert.Equal(t, diff.Region{Bounds: diff.Bounds{X: 0, Y: 0, Width: 6, Height: 3}, ChangedPixels: 7}, grouped[0])
}

func TestFilterImageRegionsRemovesTinyClusters(t *testing.T) {
	regions := []diff.Region{{ChangedPixels: 2}, {ChangedPixels: 20}, {ChangedPixels: 5}}

	filtered := filterImageRegions(regions, 5)

	assert.Equal(t, []diff.Region{{ChangedPixels: 20}, {ChangedPixels: 5}}, filtered)
}

func TestParseImageRegion(t *testing.T) {
	region, err := parseImageRegion("10, 20,300,400")

	require.NoError(t, err)
	assert.Equal(t, &diff.Bounds{X: 10, Y: 20, Width: 300, Height: 400}, region)
}

func TestDiffImageCommandRejectsArtifactPathCollisions(t *testing.T) {
	tests := []struct {
		name, output, overlay, expected string
	}{
		{name: "output is reference", output: "reference.png", expected: "--output must not overwrite an input image"},
		{name: "output is actual", output: "actual.png", expected: "--output must not overwrite an input image"},
		{name: "overlay is reference", output: "mask.png", overlay: "reference.png", expected: "--overlay must not overwrite an input image"},
		{name: "overlay is output", output: "mask.png", overlay: "./mask.png", expected: "--overlay must differ from --output"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := []string{"reference.png", "actual.png", "--output", test.output}
			if test.overlay != "" {
				args = append(args, "--overlay", test.overlay)
			}
			result := executeCommand(newCommand(diff.CompareImagesWithThresholds), args...)

			assert.EqualError(t, result.Err, test.expected)
		})
	}
}

func TestDiffImageCommandRejectsEmptyIgnoredRegions(t *testing.T) {
	for _, region := range []string{"0,0,0,1", "0,0,1,0", "0,0,-1,1", "0,0,1,-1"} {
		t.Run(region, func(t *testing.T) {
			result := executeCommand(newCommand(diff.CompareImagesWithThresholds), "reference.png", "actual.png", "--output", "mask.png", "--ignore-region", region)

			assert.EqualError(t, result.Err, "invalid --ignore-region: width and height must be positive")
		})
	}
}

func TestDiffImageCommandRejectsInvalidAnalysisLimits(t *testing.T) {
	tests := []struct {
		name, flag, value, expected string
	}{
		{name: "negative offset radius", flag: "--suggest-offset", value: "-1", expected: "--suggest-offset must be non-negative"},
		{name: "negative movement radius", flag: "--suggest-movement", value: "-1", expected: "--suggest-movement must be non-negative"},
		{name: "negative region gap", flag: "--region-gap", value: "-1", expected: "--region-gap must be non-negative"},
		{name: "zero minimum region pixels", flag: "--min-region-pixels", value: "0", expected: "--min-region-pixels must be positive"},
		{name: "negative perceptual threshold", flag: "--perceptual-threshold", value: "-0.1", expected: "--perceptual-threshold must be a finite non-negative number"},
		{name: "NaN perceptual threshold", flag: "--perceptual-threshold", value: "NaN", expected: "--perceptual-threshold must be a finite non-negative number"},
		{name: "infinite perceptual threshold", flag: "--perceptual-threshold", value: "+Inf", expected: "--perceptual-threshold must be a finite non-negative number"},
		{name: "NaN RMSE limit", flag: "--max-rmse", value: "NaN", expected: "--max-rmse must be -1 or a finite non-negative number"},
		{name: "invalid negative RMSE limit", flag: "--max-rmse", value: "-2", expected: "--max-rmse must be -1 or a finite non-negative number"},
		{name: "changed ratio above one", flag: "--max-changed-ratio", value: "1.1", expected: "--max-changed-ratio must be -1 or between 0 and 1"},
		{name: "invalid negative changed ratio", flag: "--max-changed-ratio", value: "-2", expected: "--max-changed-ratio must be -1 or between 0 and 1"},
		{name: "perceptual changed ratio above one", flag: "--max-perceptual-changed-ratio", value: "1.1", expected: "--max-perceptual-changed-ratio must be -1 or between 0 and 1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := executeCommand(newCommand(diff.CompareImagesWithThresholds), "reference.png", "actual.png", "--output", "mask.png", test.flag, test.value)

			assert.EqualError(t, result.Err, test.expected)
		})
	}
}

func TestDiffImageCommandFailsValidationThresholdWithStructuredEvidence(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	changed := image.NewRGBA(image.Rect(0, 0, 2, 2))
	changed.Set(0, 0, image.White)
	writeTestPNG(t, actual, changed)

	result := executeCommand(newCommand(diff.CompareImagesWithThresholds), reference, actual, "--output", mask, "--max-changed-ratio", "0.1", "--max-perceptual-changed-ratio", "0")

	assert.EqualError(t, result.Err, "image diff validation failed: changed ratio 0.250000 exceeds maximum 0.100000")
	var output outputEnvelope
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
	assert.Equal(t, 0.25, output.ChangedRatio)
	require.NotNil(t, output.Validation)
	assert.False(t, output.Validation.Passed)
	assert.Equal(t, []comparisonValidationFailure{
		{Metric: "changedRatio", Actual: 0.25, Maximum: 0.1},
	}, output.Validation.Failed)
	assert.Equal(t, mask, output.Mask)
}
