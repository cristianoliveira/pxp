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

	"github.com/cristianoliveira/pxp/internal/annotationio"
	"github.com/cristianoliveira/pxp/internal/annotations"
	"github.com/cristianoliveira/pxp/internal/cli"
	"github.com/cristianoliveira/pxp/internal/imagecontext"
	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/cristianoliveira/pxp/internal/imageio"
	"github.com/spf13/cobra"
)

//nolint:lll // keep this expression together
type imageComparer func(referencePath, actualPath, maskPath string, threshold uint8, perceptualThreshold float64, region *diff.Bounds, ignored []diff.Bounds) (diff.ImageComparison, error)

type comparisonOptions struct {
	configuration             *comparisonConfiguration
	output                    string
	overlay                   string
	annotations               string
	threshold                 uint8
	perceptualThreshold       float64
	maxRMSE                   float64
	maxChangedRatio           float64
	maxPerceptualChangedRatio float64
	visualContextEnabled      bool
	provider                  string
	model                     string
	visualContextPrompt       string
	offsetRadius              int
	movementRadius            int
	regionGap                 int
	minRegionPixels           int
	maxRegions                int
	full                      bool
	region                    *diff.Bounds
	regionValue               string
	ignoredValues             []string
	ignored                   []diff.Bounds
}

func runComparisonCommand(cmd *cobra.Command, args []string, compare imageComparer) error {
	referencePath, actualPath := args[0], args[1]
	options, err := readComparisonOptions(cmd, referencePath, actualPath)
	if err != nil {
		return err
	}
	inputs, ignored, err := prepareComparisonInputs(cmd, referencePath, actualPath, options.ignored)
	if err != nil {
		return err
	}
	defer inputs.cleanup()

	result, decoded, err := comparePreparedImages(
		compare,
		inputs,
		options.output,
		options.threshold,
		options.perceptualThreshold,
		options.region,
		ignored,
	)
	if err != nil {
		return err
	}
	result.Inputs = inputs.metadata

	annotationDocument, err := loadAnnotationDocument(options.annotations, inputs.referencePath)
	if err != nil {
		return err
	}
	decoded, err = writeComparisonArtifacts(options, inputs, &result, decoded, ignored)
	if err != nil {
		return err
	}
	regionCount, regionsTruncated, err := processComparisonRegions(
		options,
		inputs,
		&result,
		annotationDocument,
		decoded,
		ignored,
	)
	if err != nil {
		return err
	}

	validation, validationErr := evaluateComparisonValidation(
		result,
		options.maxRMSE,
		options.maxChangedRatio,
		options.maxPerceptualChangedRatio,
	)
	outputResult := outputEnvelope{
		ImageComparison:  result,
		RegionCount:      regionCount,
		RegionsReturned:  len(result.Regions),
		RegionsTruncated: regionsTruncated,
		Configuration:    options.configuration,
		Validation:       validation,
	}
	return writeComparisonOutput(
		cmd,
		args,
		options,
		inputs,
		outputResult,
		validationErr,
	)
}

func readComparisonOptions(
	cmd *cobra.Command,
	referencePath, actualPath string,
) (comparisonOptions, error) {
	configuration, err := applyComparisonProfile(cmd)
	if err != nil {
		return comparisonOptions{}, err
	}
	output, _ := cmd.Flags().GetString("output")
	overlay, _ := cmd.Flags().GetString("overlay")
	annotationsPath, _ := cmd.Flags().GetString("annotations")
	threshold, _ := cmd.Flags().GetUint8("threshold")
	perceptualThreshold, _ := cmd.Flags().GetFloat64("perceptual-threshold")
	maxRMSE, _ := cmd.Flags().GetFloat64("max-rmse")
	maxChangedRatio, _ := cmd.Flags().GetFloat64("max-changed-ratio")
	maxPerceptualChangedRatio, _ := cmd.Flags().GetFloat64("max-perceptual-changed-ratio")
	visualContextEnabled, _ := cmd.Flags().GetBool("visual-context")
	provider, _ := cmd.Flags().GetString("visual-context-provider")
	model, _ := cmd.Flags().GetString("visual-context-model")
	visualContextPrompt, _ := cmd.Flags().GetString("visual-context-prompt")
	offsetRadius, _ := cmd.Flags().GetInt("suggest-offset")
	movementRadius, _ := cmd.Flags().GetInt("suggest-movement")
	regionGap, _ := cmd.Flags().GetInt("region-gap")
	minRegionPixels, _ := cmd.Flags().GetInt("min-region-pixels")
	maxRegions, _ := cmd.Flags().GetInt("max-regions")
	full, _ := cmd.Flags().GetBool("full")
	regionValue := cmd.Flags().Lookup("region").Value.String()
	ignoredValues, _ := cmd.Flags().GetStringArray("ignore-region")
	options := comparisonOptions{
		configuration:             configuration,
		output:                    output,
		overlay:                   overlay,
		annotations:               annotationsPath,
		threshold:                 threshold,
		perceptualThreshold:       perceptualThreshold,
		maxRMSE:                   maxRMSE,
		maxChangedRatio:           maxChangedRatio,
		maxPerceptualChangedRatio: maxPerceptualChangedRatio,
		visualContextEnabled:      visualContextEnabled,
		provider:                  provider,
		model:                     model,
		visualContextPrompt:       visualContextPrompt,
		offsetRadius:              offsetRadius,
		movementRadius:            movementRadius,
		regionGap:                 regionGap,
		minRegionPixels:           minRegionPixels,
		maxRegions:                maxRegions,
		full:                      full,
		regionValue:               regionValue,
		ignoredValues:             ignoredValues,
	}
	if options.output == "" {
		options.output = defaultMaskPath(actualPath)
	}
	if err := validateComparisonOptions(&options, referencePath, actualPath); err != nil {
		return comparisonOptions{}, cli.NewUsageError(err)
	}
	return options, nil
}

func validateComparisonOptions(options *comparisonOptions, referencePath, actualPath string) error {
	if err := validateVisualContextProvider(
		options.visualContextEnabled,
		options.provider,
	); err != nil {
		return err
	}
	if err := validateComparisonArtifactPaths(
		referencePath, actualPath, options.output, options.overlay,
	); err != nil {
		return err
	}
	if err := validateRegionControls(
		options.offsetRadius,
		options.movementRadius,
		options.regionGap,
		options.minRegionPixels,
		options.maxRegions,
	); err != nil {
		return err
	}
	if err := validateComparisonThresholds(
		options.perceptualThreshold,
		options.maxRMSE,
		options.maxChangedRatio,
		options.maxPerceptualChangedRatio,
	); err != nil {
		return err
	}
	var err error
	options.region, err = parseImageRegion(options.regionValue)
	if err != nil {
		return err
	}
	options.ignored, err = parseIgnoredRegions(options.ignoredValues)
	return err
}

func loadAnnotationDocument(path, referencePath string) (*annotations.Document, error) {
	if path == "" {
		return nil, nil
	}
	document, err := annotationio.Load(path)
	if err != nil {
		return nil, err
	}
	imageWidth, imageHeight, err := imageio.PNGDimensions(referencePath)
	if err != nil {
		return nil, err
	}
	if err := document.ValidateDimensions(imageWidth, imageHeight); err != nil {
		return nil, err
	}
	return document, nil
}

func writeComparisonArtifacts(
	options comparisonOptions,
	inputs preparedImageInputs,
	result *diff.ImageComparison,
	decoded *diff.DecodedImages,
	ignored []diff.Bounds,
) (*diff.DecodedImages, error) {
	if options.overlay == "" && options.offsetRadius <= 0 {
		return decoded, nil
	}
	if err := ensureDecodedImages(&decoded, inputs); err != nil {
		return nil, err
	}
	if options.overlay != "" {
		overlayImage, err := decoded.Overlay(options.region, ignored)
		if err != nil {
			return nil, err
		}
		if err := imageio.WritePNG(options.overlay, overlayImage); err != nil {
			return nil, err
		}
		result.Overlay = options.overlay
	}
	if options.offsetRadius > 0 {
		suggestedOffset := decoded.SuggestOffset(options.offsetRadius, options.region, ignored)
		if !math.IsInf(suggestedOffset.RMSE, 0) && !math.IsNaN(suggestedOffset.RMSE) {
			result.SuggestedOffset = &suggestedOffset
		}
	}
	return decoded, nil
}

func ensureDecodedImages(decoded **diff.DecodedImages, inputs preparedImageInputs) error {
	if *decoded != nil {
		return nil
	}
	var err error
	*decoded, err = imageio.LoadDecodedImages(inputs.referencePath, inputs.actualPath)
	return err
}

func processComparisonRegions(
	options comparisonOptions,
	inputs preparedImageInputs,
	result *diff.ImageComparison,
	annotationDocument *annotations.Document,
	decoded *diff.DecodedImages,
	ignored []diff.Bounds,
) (int, bool, error) {
	result.Regions = groupImageRegions(result.Regions, options.regionGap)
	result.Regions = filterImageRegions(result.Regions, options.minRegionPixels)
	var regionCount int
	var regionsTruncated bool
	result.Regions, regionCount, regionsTruncated = limitImageRegions(
		result.Regions,
		options.maxRegions,
		options.full,
	)
	regionMetrics, decoded, err := measureRegionMetrics(
		options,
		inputs,
		result.Regions,
		decoded,
		ignored,
	)
	if err != nil {
		return 0, false, err
	}
	for index := range result.Regions {
		region := &result.Regions[index]
		region.InputBounds = inputBounds(region.Bounds, inputs.metadata)
		if annotationDocument != nil {
			region.Annotations = annotationDocument.Intersections(annotations.Bounds{
				X: region.Bounds.X, Y: region.Bounds.Y,
				Width: region.Bounds.Width, Height: region.Bounds.Height,
			})
		}
		applyRegionMetrics(region, regionMetrics[index])
	}
	if options.movementRadius > 0 && len(result.Regions) > 0 {
		regionBounds := make([]diff.Bounds, len(result.Regions))
		for index := range result.Regions {
			regionBounds[index] = result.Regions[index].Bounds
		}
		result.MovedRegions = decoded.SuggestRegionMovements(
			regionBounds, options.movementRadius, ignored,
		)
	}
	return regionCount, regionsTruncated, nil
}

func measureRegionMetrics(
	options comparisonOptions,
	inputs preparedImageInputs,
	regions []diff.Region,
	decoded *diff.DecodedImages,
	ignored []diff.Bounds,
) ([]diff.RegionMetrics, *diff.DecodedImages, error) {
	metrics := make([]diff.RegionMetrics, len(regions))
	if len(regions) == 0 {
		return metrics, decoded, nil
	}
	if err := ensureDecodedImages(&decoded, inputs); err != nil {
		return nil, nil, err
	}
	regionBounds := make([]diff.Bounds, len(regions))
	for index := range regions {
		regionBounds[index] = regions[index].Bounds
	}
	metrics, err := decoded.MeasureRegions(
		regionBounds,
		options.threshold,
		options.perceptualThreshold,
		ignored,
	)
	return metrics, decoded, err
}

func applyRegionMetrics(region *diff.Region, metrics diff.RegionMetrics) {
	region.ChangedPixels = metrics.ChangedPixels
	region.ChangedRatio = metrics.ChangedRatio
	region.RMSE = metrics.RMSE
	region.EdgeRMSE = metrics.EdgeRMSE
	region.PerceptualRMSE = metrics.PerceptualRMSE
	region.PerceptualChangedPixels = metrics.PerceptualChangedPixels
	region.PerceptualChangedRatio = metrics.PerceptualChangedRatio
	region.AntialiasedPixels = metrics.AntialiasedPixels
	region.DominantColorPairs = metrics.DominantColorPairs
	region.Classification = diff.ClassifyImageRegion(metrics)
}

func writeComparisonOutput(
	cmd *cobra.Command,
	args []string,
	options comparisonOptions,
	inputs preparedImageInputs,
	outputResult outputEnvelope,
	validationErr error,
) error {
	if outputResult.RegionsTruncated {
		outputResult.Hint = cli.FullHint(cmd, args)
	}
	if validationErr != nil {
		if err := writeStructured(cmd, outputResult); err != nil {
			return err
		}
		return cli.NewResultError(validationErr)
	}
	if err := addVisualContext(
		options.visualContextEnabled,
		options.provider,
		options.model,
		options.visualContextPrompt,
		inputs,
		outputResult.ImageComparison,
		&outputResult,
	); err != nil {
		return err
	}
	return writeStructured(cmd, outputResult)
}

func comparePreparedImages(
	compare imageComparer,
	inputs preparedImageInputs,
	output string,
	threshold uint8,
	perceptualThreshold float64,
	region *diff.Bounds,
	ignored []diff.Bounds,
) (diff.ImageComparison, *diff.DecodedImages, error) {
	if compare != nil {
		result, err := compare(
			inputs.referencePath,
			inputs.actualPath,
			output,
			threshold,
			perceptualThreshold,
			region,
			ignored,
		)
		return result, nil, err
	}
	decoded, err := imageio.LoadDecodedImages(inputs.referencePath, inputs.actualPath)
	if err != nil {
		return diff.ImageComparison{}, nil, err
	}
	result, mask, err := decoded.Compare(threshold, perceptualThreshold, region, ignored)
	if err != nil {
		return diff.ImageComparison{}, nil, err
	}
	if output != "" {
		if err := imageio.WritePNG(output, mask); err != nil {
			return diff.ImageComparison{}, nil, err
		}
		result.Mask = output
	}
	return result, decoded, nil
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

func validateRegionControls(
	offsetRadius, movementRadius, regionGap, minRegionPixels, maxRegions int,
) error {
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

func validateComparisonThresholds(
	perceptualThreshold, maxRMSE, maxChangedRatio, maxPerceptualChangedRatio float64,
) error {
	if perceptualThreshold < 0 || math.IsNaN(perceptualThreshold) ||
		math.IsInf(perceptualThreshold, 0) {
		return fmt.Errorf("--perceptual-threshold must be a finite non-negative number")
	}
	if maxRMSE != -1 && (maxRMSE < 0 || math.IsNaN(maxRMSE) || math.IsInf(maxRMSE, 0)) {
		return fmt.Errorf("--max-rmse must be -1 or a finite non-negative number")
	}
	if maxChangedRatio != -1 &&
		//nolint:lll // keep this expression together
		(maxChangedRatio < 0 || maxChangedRatio > 1 || math.IsNaN(maxChangedRatio) || math.IsInf(maxChangedRatio, 0)) {
		return fmt.Errorf("--max-changed-ratio must be -1 or between 0 and 1")
	}
	if maxPerceptualChangedRatio != -1 &&
		//nolint:lll // keep this expression together
		(maxPerceptualChangedRatio < 0 || maxPerceptualChangedRatio > 1 || math.IsNaN(maxPerceptualChangedRatio) || math.IsInf(maxPerceptualChangedRatio, 0)) {
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

func validateComparisonArtifactPaths(
	referencePath, actualPath, outputPath, overlayPath string,
) error {
	if outputPath != "" &&
		(samePath(outputPath, referencePath) || samePath(outputPath, actualPath)) {
		return fmt.Errorf("--output must not overwrite an input image")
	}
	if overlayPath != "" &&
		(samePath(overlayPath, referencePath) || samePath(overlayPath, actualPath)) {
		return fmt.Errorf("--overlay must not overwrite an input image")
	}
	if overlayPath != "" && outputPath != "" && samePath(overlayPath, outputPath) {
		return fmt.Errorf("--overlay must differ from --output")
	}
	return nil
}

func addVisualContext(
	enabled bool,
	provider, model, prompt string,
	inputs preparedImageInputs,
	result diff.ImageComparison,
	output *outputEnvelope,
) error {
	if !enabled {
		return nil
	}
	config, err := imagecontext.LoadProviderConfig(provider, model)
	if errors.Is(err, imagecontext.ErrNotConfigured) {
		output.VisualContext = &imagecontext.Result{
			Provider: provider,
			Advisory: true,
			Disclaimer: fmt.Sprintf(
				//nolint:lll // keep this expression together
				"Visual context unavailable: configure %s credentials in the Pi Spectacles config or environment.",
				provider,
			),
		}
		return nil
	}
	if err != nil {
		return err
	}
	regions := make([]imagecontext.Region, len(result.Regions))
	for index, region := range result.Regions {
		regions[index] = imagecontext.Region{
			ID: fmt.Sprintf("r%d", index+1),
			Bounds: imagecontext.Bounds{
				X:      region.Bounds.X,
				Y:      region.Bounds.Y,
				Width:  region.Bounds.Width,
				Height: region.Bounds.Height,
			},
		}
	}
	client, err := imagecontext.NewClient(provider, config)
	if err != nil {
		return err
	}
	visualContext, err := client.Describe(
		context.Background(),
		imagecontext.Input{
			ReferencePath: inputs.referencePath,
			ActualPath:    inputs.actualPath,
			Regions:       regions,
			Prompt:        prompt,
		},
	)
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
	sort.SliceStable(
		grouped,
		func(i, j int) bool { return grouped[i].ChangedPixels > grouped[j].ChangedPixels },
	)
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
	return diff.Region{
		Bounds:        diff.Bounds{X: left, Y: top, Width: right - left, Height: bottom - top},
		ChangedPixels: first.ChangedPixels + second.ChangedPixels,
	}
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

func evaluateComparisonValidation(
	result diff.ImageComparison,
	maxRMSE, maxChangedRatio, maxPerceptualChangedRatio float64,
) (*comparisonValidation, error) {
	configured := maxRMSE >= 0 || maxChangedRatio >= 0 || maxPerceptualChangedRatio >= 0
	if !configured {
		return nil, nil
	}

	failed := make([]comparisonValidationFailure, 0, 3)
	if maxRMSE >= 0 && result.RMSE > maxRMSE {
		failed = append(
			failed,
			comparisonValidationFailure{Metric: "rmse", Actual: result.RMSE, Maximum: maxRMSE},
		)
	}
	if maxChangedRatio >= 0 && result.ChangedRatio > maxChangedRatio {
		failed = append(
			failed,
			comparisonValidationFailure{
				Metric:  "changedRatio",
				Actual:  result.ChangedRatio,
				Maximum: maxChangedRatio,
			},
		)
	}
	if maxPerceptualChangedRatio >= 0 && result.PerceptualChangedRatio > maxPerceptualChangedRatio {
		failed = append(
			failed,
			comparisonValidationFailure{
				Metric:  "perceptualChangedRatio",
				Actual:  result.PerceptualChangedRatio,
				Maximum: maxPerceptualChangedRatio,
			},
		)
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
	return validation, fmt.Errorf(
		"image diff validation failed: %s %.6f exceeds maximum %.6f",
		label,
		failure.Actual,
		failure.Maximum,
	)
}

func prepareComparisonInputs(
	command *cobra.Command,
	referencePath, actualPath string,
	ignored []diff.Bounds,
) (preparedImageInputs, []diff.Bounds, error) {
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
		return preparedImageInputs{}, nil, cli.NewUsageError(
			fmt.Errorf("--reference-crop and --reference-metadata cannot be used together"),
		)
	}
	referenceMetadata, err := loadExportMetadata(referenceMetadataPath)
	if err != nil {
		return preparedImageInputs{}, nil, err
	}
	inputs, err := prepareImageInputs(
		referencePath,
		actualPath,
		referenceCrop,
		actualCrop,
		referenceMetadata,
	)
	if err != nil {
		return preparedImageInputs{}, nil, err
	}
	comparisonMask, _ := command.Flags().GetString("mask")
	if comparisonMask == "" {
		return inputs, ignored, nil
	}
	maskedRegions, err := imageio.IgnoredRegionsFromMask(comparisonMask, inputs.referencePath)
	if err != nil {
		inputs.cleanup()
		return preparedImageInputs{}, nil, err
	}
	return inputs, append(ignored, maskedRegions...), nil
}
