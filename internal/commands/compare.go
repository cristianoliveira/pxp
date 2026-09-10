package commands

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cristianoliveira/pxp/internal/annotations"
	"github.com/cristianoliveira/pxp/internal/cli"
	"github.com/cristianoliveira/pxp/internal/imagecontext"
	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	reportpkg "github.com/cristianoliveira/pxp/internal/report"
	"github.com/spf13/cobra"
)

type imageComparer func(referencePath, actualPath, maskPath string, threshold uint8, perceptualThreshold float64, region *diff.Bounds, ignored []diff.Bounds) (diff.ImageComparison, error)

func runComparisonCommand(cmd *cobra.Command, args []string, compare imageComparer) error {
	configuration, err := applyComparisonProfile(cmd)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")
	if output == "" {
		output = defaultMaskPath(args[1])
	}
	threshold, _ := cmd.Flags().GetUint8("threshold")
	overlay, _ := cmd.Flags().GetString("overlay")
	report, _ := cmd.Flags().GetString("report")
	visualContextEnabled, _ := cmd.Flags().GetBool("visual-context")
	provider, _ := cmd.Flags().GetString("visual-context-provider")
	if err := validateVisualContextProvider(visualContextEnabled, provider); err != nil {
		return cli.NewUsageError(err)
	}
	if err := validateComparisonArtifactPaths(args[0], args[1], output, overlay, report); err != nil {
		return cli.NewUsageError(err)
	}
	offsetRadius, _ := cmd.Flags().GetInt("suggest-offset")
	movementRadius, _ := cmd.Flags().GetInt("suggest-movement")
	regionGap, _ := cmd.Flags().GetInt("region-gap")
	minRegionPixels, _ := cmd.Flags().GetInt("min-region-pixels")
	maxRegions, _ := cmd.Flags().GetInt("max-regions")
	if err := validateRegionControls(offsetRadius, movementRadius, regionGap, minRegionPixels, maxRegions); err != nil {
		return cli.NewUsageError(err)
	}
	full, _ := cmd.Flags().GetBool("full")
	perceptualThreshold, _ := cmd.Flags().GetFloat64("perceptual-threshold")
	maxRMSE, _ := cmd.Flags().GetFloat64("max-rmse")
	maxChangedRatio, _ := cmd.Flags().GetFloat64("max-changed-ratio")
	maxPerceptualChangedRatio, _ := cmd.Flags().GetFloat64("max-perceptual-changed-ratio")
	if err := validateComparisonThresholds(perceptualThreshold, maxRMSE, maxChangedRatio, maxPerceptualChangedRatio); err != nil {
		return cli.NewUsageError(err)
	}
	region, err := parseImageRegion(cmd.Flags().Lookup("region").Value.String())
	if err != nil {
		return cli.NewUsageError(err)
	}
	ignoredValues, _ := cmd.Flags().GetStringArray("ignore-region")
	ignored, err := parseIgnoredRegions(ignoredValues)
	if err != nil {
		return cli.NewUsageError(err)
	}
	inputs, ignored, err := prepareComparisonInputs(cmd, args, ignored)
	if err != nil {
		return err
	}
	defer inputs.cleanup()
	result, decoded, err := comparePreparedImages(compare, inputs, output, threshold, perceptualThreshold, region, ignored)
	if err != nil {
		return err
	}
	result.Inputs = inputs.metadata
	annotationsPath, _ := cmd.Flags().GetString("annotations")
	var annotationDocument *annotations.Document
	if annotationsPath != "" {
		annotationDocument, err = annotations.Load(annotationsPath)
		if err != nil {
			return err
		}
		imageWidth, imageHeight, dimensionsErr := diff.PNGDimensions(inputs.referencePath)
		if dimensionsErr != nil {
			return dimensionsErr
		}
		if err := annotationDocument.ValidateDimensions(imageWidth, imageHeight); err != nil {
			return err
		}
	}
	if overlay != "" {
		if decoded == nil {
			decoded, err = diff.LoadDecodedImages(inputs.referencePath, inputs.actualPath)
			if err != nil {
				return err
			}
		}
		if err := decoded.WriteOverlay(overlay, region, ignored); err != nil {
			return err
		}
		result.Overlay = overlay
	}
	if offsetRadius > 0 {
		if decoded == nil {
			decoded, err = diff.LoadDecodedImages(inputs.referencePath, inputs.actualPath)
			if err != nil {
				return err
			}
		}
		suggestedOffset := decoded.SuggestOffset(offsetRadius, region, ignored)
		if !math.IsInf(suggestedOffset.RMSE, 0) && !math.IsNaN(suggestedOffset.RMSE) {
			result.SuggestedOffset = &suggestedOffset
		}
	}
	result.Regions = groupImageRegions(result.Regions, regionGap)
	result.Regions = filterImageRegions(result.Regions, minRegionPixels)
	var regionCount int
	var regionsTruncated bool
	result.Regions, regionCount, regionsTruncated = limitImageRegions(result.Regions, maxRegions, full)
	regionMetrics := make([]diff.RegionMetrics, len(result.Regions))
	if len(result.Regions) > 0 {
		regionBounds := make([]diff.Bounds, len(result.Regions))
		for index := range result.Regions {
			regionBounds[index] = result.Regions[index].Bounds
		}
		if decoded == nil {
			decoded, err = diff.LoadDecodedImages(inputs.referencePath, inputs.actualPath)
			if err != nil {
				return err
			}
		}
		regionMetrics, err = decoded.MeasureRegions(regionBounds, threshold, perceptualThreshold, ignored)
		if err != nil {
			return err
		}
	}
	for index := range result.Regions {
		result.Regions[index].InputBounds = inputBounds(result.Regions[index].Bounds, inputs.metadata)
		if annotationDocument != nil {
			bounds := result.Regions[index].Bounds
			result.Regions[index].Annotations = annotationDocument.Intersections(annotations.Bounds{X: bounds.X, Y: bounds.Y, Width: bounds.Width, Height: bounds.Height})
		}
		metrics := regionMetrics[index]
		result.Regions[index].ChangedPixels = metrics.ChangedPixels
		result.Regions[index].ChangedRatio = metrics.ChangedRatio
		result.Regions[index].RMSE = metrics.RMSE
		result.Regions[index].EdgeRMSE = metrics.EdgeRMSE
		result.Regions[index].PerceptualRMSE = metrics.PerceptualRMSE
		result.Regions[index].PerceptualChangedPixels = metrics.PerceptualChangedPixels
		result.Regions[index].PerceptualChangedRatio = metrics.PerceptualChangedRatio
		result.Regions[index].AntialiasedPixels = metrics.AntialiasedPixels
		result.Regions[index].DominantColorPairs = metrics.DominantColorPairs
		result.Regions[index].Classification = diff.ClassifyImageRegion(metrics)
	}
	if movementRadius > 0 && len(result.Regions) > 0 {
		regionBounds := make([]diff.Bounds, len(result.Regions))
		for index := range result.Regions {
			regionBounds[index] = result.Regions[index].Bounds
		}
		result.MovedRegions = decoded.SuggestRegionMovements(regionBounds, movementRadius, ignored)
	}
	validation, validationErr := evaluateComparisonValidation(result, maxRMSE, maxChangedRatio, maxPerceptualChangedRatio)
	outputResult := outputEnvelope{
		ImageComparison:  result,
		RegionCount:      regionCount,
		RegionsReturned:  len(result.Regions),
		RegionsTruncated: regionsTruncated,
		Configuration:    configuration,
		Validation:       validation,
	}
	if regionsTruncated {
		outputResult.Hint = cli.FullHint(cmd, args)
	}
	if validationErr != nil {
		if err := writeStructured(cmd, outputResult); err != nil {
			return err
		}
		return cli.NewResultError(validationErr)
	}
	if err := writeComparisonReport(report, inputs, output, overlay, threshold, perceptualThreshold, region, result); err != nil {
		return err
	}
	model, _ := cmd.Flags().GetString("visual-context-model")
	visualContextPrompt, _ := cmd.Flags().GetString("visual-context-prompt")
	if err := addVisualContext(visualContextEnabled, provider, model, visualContextPrompt, inputs, result, &outputResult); err != nil {
		return err
	}
	return writeStructured(cmd, outputResult)
}

func comparePreparedImages(compare imageComparer, inputs preparedImageInputs, output string, threshold uint8, perceptualThreshold float64, region *diff.Bounds, ignored []diff.Bounds) (diff.ImageComparison, *diff.DecodedImages, error) {
	if compare != nil {
		result, err := compare(inputs.referencePath, inputs.actualPath, output, threshold, perceptualThreshold, region, ignored)
		return result, nil, err
	}
	decoded, err := diff.LoadDecodedImages(inputs.referencePath, inputs.actualPath)
	if err != nil {
		return diff.ImageComparison{}, nil, err
	}
	result, err := decoded.Compare(output, threshold, perceptualThreshold, region, ignored)
	return result, decoded, err
}

func parseIgnoredRegions(values []string) ([]diff.Bounds, error) {
	ignored := make([]diff.Bounds, 0, len(values))
	for _, value := range values {
		region, err := parseImageRegion(value)
		if err != nil {
			return nil, fmt.Errorf("invalid --ignore-region: %w", err)
		}
		if region.Width < 1 || region.Height < 1 {
			return nil, fmt.Errorf("invalid --ignore-region: width and height must be positive")
		}
		ignored = append(ignored, *region)
	}
	return ignored, nil
}

func validateRegionControls(offsetRadius, movementRadius, regionGap, minRegionPixels, maxRegions int) error {
	if offsetRadius < 0 {
		return fmt.Errorf("--suggest-offset must be non-negative")
	}
	if movementRadius < 0 {
		return fmt.Errorf("--suggest-movement must be non-negative")
	}
	if regionGap < 0 {
		return fmt.Errorf("--region-gap must be non-negative")
	}
	if minRegionPixels < 1 {
		return fmt.Errorf("--min-region-pixels must be positive")
	}
	if maxRegions < 1 {
		return fmt.Errorf("--max-regions must be positive")
	}
	return nil
}

func validateComparisonThresholds(perceptualThreshold, maxRMSE, maxChangedRatio, maxPerceptualChangedRatio float64) error {
	if perceptualThreshold < 0 || math.IsNaN(perceptualThreshold) || math.IsInf(perceptualThreshold, 0) {
		return fmt.Errorf("--perceptual-threshold must be a finite non-negative number")
	}
	if maxRMSE != -1 && (maxRMSE < 0 || math.IsNaN(maxRMSE) || math.IsInf(maxRMSE, 0)) {
		return fmt.Errorf("--max-rmse must be -1 or a finite non-negative number")
	}
	if maxChangedRatio != -1 && (maxChangedRatio < 0 || maxChangedRatio > 1 || math.IsNaN(maxChangedRatio) || math.IsInf(maxChangedRatio, 0)) {
		return fmt.Errorf("--max-changed-ratio must be -1 or between 0 and 1")
	}
	if maxPerceptualChangedRatio != -1 && (maxPerceptualChangedRatio < 0 || maxPerceptualChangedRatio > 1 || math.IsNaN(maxPerceptualChangedRatio) || math.IsInf(maxPerceptualChangedRatio, 0)) {
		return fmt.Errorf("--max-perceptual-changed-ratio must be -1 or between 0 and 1")
	}
	return nil
}

func validateVisualContextProvider(enabled bool, provider string) error {
	if !enabled || provider == "openrouter" || provider == "openai" {
		return nil
	}
	return fmt.Errorf("unsupported visual context provider %q", provider)
}

func validateComparisonArtifactPaths(referencePath, actualPath, outputPath, overlayPath, reportPath string) error {
	if outputPath != "" && (samePath(outputPath, referencePath) || samePath(outputPath, actualPath)) {
		return fmt.Errorf("--output must not overwrite an input image")
	}
	if overlayPath != "" && (samePath(overlayPath, referencePath) || samePath(overlayPath, actualPath)) {
		return fmt.Errorf("--overlay must not overwrite an input image")
	}
	if overlayPath != "" && outputPath != "" && samePath(overlayPath, outputPath) {
		return fmt.Errorf("--overlay must differ from --output")
	}
	if reportPath != "" && (samePath(reportPath, referencePath) || samePath(reportPath, actualPath) || (outputPath != "" && samePath(reportPath, outputPath)) || samePath(reportPath, overlayPath)) {
		return fmt.Errorf("--report must not overwrite an input, mask, or overlay")
	}
	return nil
}

func writeComparisonReport(report string, inputs preparedImageInputs, maskPath, overlayPath string, threshold uint8, perceptualThreshold float64, region *diff.Bounds, result diff.ImageComparison) error {
	if report == "" {
		return nil
	}
	return reportpkg.Write(report, reportpkg.Input{
		ReferencePath:       inputs.referencePath,
		ActualPath:          inputs.actualPath,
		MaskPath:            maskPath,
		OverlayPath:         overlayPath,
		Threshold:           threshold,
		PerceptualThreshold: perceptualThreshold,
		ComparedRegion:      region,
		Result:              result,
	})
}

func addVisualContext(enabled bool, provider, model, prompt string, inputs preparedImageInputs, result diff.ImageComparison, output *outputEnvelope) error {
	if !enabled {
		return nil
	}
	config, err := imagecontext.LoadProviderConfig(provider, model)
	if errors.Is(err, imagecontext.ErrNotConfigured) {
		output.VisualContext = &imagecontext.Result{Provider: provider, Advisory: true, Disclaimer: fmt.Sprintf("Visual context unavailable: configure %s credentials in the Pi Spectacles config or environment.", provider)}
		return nil
	}
	if err != nil {
		return err
	}
	regions := make([]imagecontext.Region, len(result.Regions))
	for index, region := range result.Regions {
		regions[index] = imagecontext.Region{ID: fmt.Sprintf("r%d", index+1), Bounds: imagecontext.Bounds{X: region.Bounds.X, Y: region.Bounds.Y, Width: region.Bounds.Width, Height: region.Bounds.Height}}
	}
	client, err := imagecontext.NewClient(provider, config)
	if err != nil {
		return err
	}
	visualContext, err := client.Describe(context.Background(), imagecontext.Input{ReferencePath: inputs.referencePath, ActualPath: inputs.actualPath, Regions: regions, Prompt: prompt})
	if err != nil {
		return err
	}
	output.VisualContext = &visualContext
	return nil
}

func defaultMaskPath(actualPath string) string {
	extension := filepath.Ext(actualPath)
	if extension == "" {
		return actualPath + ".diff.png"
	}
	return strings.TrimSuffix(actualPath, extension) + ".diff.png"
}

func groupImageRegions(regions []diff.Region, gap int) []diff.Region {
	grouped := append([]diff.Region(nil), regions...)
	for merged := true; merged; {
		merged = false
		for i := 0; i < len(grouped) && !merged; i++ {
			for j := i + 1; j < len(grouped); j++ {
				if !regionsWithinGap(grouped[i].Bounds, grouped[j].Bounds, gap) {
					continue
				}
				grouped[i] = mergeImageRegions(grouped[i], grouped[j])
				grouped = append(grouped[:j], grouped[j+1:]...)
				merged = true
				break
			}
		}
	}
	sort.SliceStable(grouped, func(i, j int) bool { return grouped[i].ChangedPixels > grouped[j].ChangedPixels })
	return grouped
}

func regionsWithinGap(first, second diff.Bounds, gap int) bool {
	return first.X <= second.X+second.Width+gap && second.X <= first.X+first.Width+gap &&
		first.Y <= second.Y+second.Height+gap && second.Y <= first.Y+first.Height+gap
}

func mergeImageRegions(first, second diff.Region) diff.Region {
	left, top := min(first.Bounds.X, second.Bounds.X), min(first.Bounds.Y, second.Bounds.Y)
	right := max(first.Bounds.X+first.Bounds.Width, second.Bounds.X+second.Bounds.Width)
	bottom := max(first.Bounds.Y+first.Bounds.Height, second.Bounds.Y+second.Bounds.Height)
	return diff.Region{Bounds: diff.Bounds{X: left, Y: top, Width: right - left, Height: bottom - top}, ChangedPixels: first.ChangedPixels + second.ChangedPixels}
}

func filterImageRegions(regions []diff.Region, minimumPixels int) []diff.Region {
	filtered := make([]diff.Region, 0, len(regions))
	for _, region := range regions {
		if region.ChangedPixels >= minimumPixels {
			filtered = append(filtered, region)
		}
	}
	return filtered
}

func limitImageRegions(regions []diff.Region, maximum int, full bool) ([]diff.Region, int, bool) {
	count := len(regions)
	if full || count <= maximum {
		return regions, count, false
	}
	return regions[:maximum], count, true
}

func parseImageRegion(value string) (*diff.Bounds, error) {
	if value == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("--region must be x,y,width,height")
	}
	values := make([]int, 4)
	for index, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("--region must contain integers: %w", err)
		}
		values[index] = value
	}
	return &diff.Bounds{X: values[0], Y: values[1], Width: values[2], Height: values[3]}, nil
}

func samePath(first, second string) bool {
	firstAbsolute, firstErr := filepath.Abs(first)
	secondAbsolute, secondErr := filepath.Abs(second)
	if firstErr != nil || secondErr != nil {
		return filepath.Clean(first) == filepath.Clean(second)
	}
	return filepath.Clean(firstAbsolute) == filepath.Clean(secondAbsolute)
}

type comparisonValidationFailure struct {
	Metric  string  `json:"metric"`
	Actual  float64 `json:"actual"`
	Maximum float64 `json:"maximum"`
}

type comparisonValidation struct {
	Passed bool                          `json:"passed"`
	Failed []comparisonValidationFailure `json:"failed"`
}

type outputEnvelope struct {
	diff.ImageComparison
	RegionCount      int                      `json:"regionCount,omitempty"`
	RegionsReturned  int                      `json:"regionsReturned,omitempty"`
	RegionsTruncated bool                     `json:"regionsTruncated,omitempty"`
	Hint             string                   `json:"hint,omitempty"`
	VisualContext    *imagecontext.Result     `json:"visualContext,omitempty"`
	Configuration    *comparisonConfiguration `json:"configuration,omitempty"`
	Validation       *comparisonValidation    `json:"validation,omitempty"`
}

func evaluateComparisonValidation(result diff.ImageComparison, maxRMSE, maxChangedRatio, maxPerceptualChangedRatio float64) (*comparisonValidation, error) {
	configured := maxRMSE >= 0 || maxChangedRatio >= 0 || maxPerceptualChangedRatio >= 0
	if !configured {
		return nil, nil
	}

	failed := make([]comparisonValidationFailure, 0, 3)
	if maxRMSE >= 0 && result.RMSE > maxRMSE {
		failed = append(failed, comparisonValidationFailure{Metric: "rmse", Actual: result.RMSE, Maximum: maxRMSE})
	}
	if maxChangedRatio >= 0 && result.ChangedRatio > maxChangedRatio {
		failed = append(failed, comparisonValidationFailure{Metric: "changedRatio", Actual: result.ChangedRatio, Maximum: maxChangedRatio})
	}
	if maxPerceptualChangedRatio >= 0 && result.PerceptualChangedRatio > maxPerceptualChangedRatio {
		failed = append(failed, comparisonValidationFailure{Metric: "perceptualChangedRatio", Actual: result.PerceptualChangedRatio, Maximum: maxPerceptualChangedRatio})
	}

	validation := &comparisonValidation{Passed: len(failed) == 0, Failed: failed}
	if validation.Passed {
		return validation, nil
	}

	failure := failed[0]
	label := map[string]string{
		"rmse":                   "RMSE",
		"changedRatio":           "changed ratio",
		"perceptualChangedRatio": "perceptual changed ratio",
	}[failure.Metric]
	return validation, fmt.Errorf("image diff validation failed: %s %.6f exceeds maximum %.6f", label, failure.Actual, failure.Maximum)
}
func prepareComparisonInputs(command *cobra.Command, args []string, ignored []diff.Bounds) (preparedImageInputs, []diff.Bounds, error) {
	referenceCrop, err := parseOptionalCrop(command, "reference-crop")
	if err != nil {
		return preparedImageInputs{}, nil, cli.NewUsageError(err)
	}
	actualCrop, err := parseOptionalCrop(command, "actual-crop")
	if err != nil {
		return preparedImageInputs{}, nil, cli.NewUsageError(err)
	}
	referenceMetadataPath, _ := command.Flags().GetString("reference-metadata")
	if referenceCrop != nil && referenceMetadataPath != "" {
		return preparedImageInputs{}, nil, cli.NewUsageError(fmt.Errorf("--reference-crop and --reference-metadata cannot be used together"))
	}
	referenceMetadata, err := loadExportMetadata(referenceMetadataPath)
	if err != nil {
		return preparedImageInputs{}, nil, err
	}
	inputs, err := prepareImageInputs(args[0], args[1], referenceCrop, actualCrop, referenceMetadata)
	if err != nil {
		return preparedImageInputs{}, nil, err
	}
	comparisonMask, _ := command.Flags().GetString("mask")
	if comparisonMask == "" {
		return inputs, ignored, nil
	}
	maskedRegions, err := diff.IgnoredRegionsFromMask(comparisonMask, inputs.referencePath)
	if err != nil {
		inputs.cleanup()
		return preparedImageInputs{}, nil, err
	}
	return inputs, append(ignored, maskedRegions...), nil
}
