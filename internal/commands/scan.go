package commands

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/cristianoliveira/pxp/internal/cli"
	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/cristianoliveira/pxp/internal/imageio"
	outputpkg "github.com/cristianoliveira/pxp/internal/output"
	"github.com/spf13/cobra"
)

type scanOutput struct {
	Axis      string            `json:"axis"`
	Index     int               `json:"index"`
	Length    int               `json:"length"`
	Total     int               `json:"total"`
	Returned  int               `json:"returned"`
	Truncated bool              `json:"truncated"`
	Hint      string            `json:"hint,omitempty"`
	Reference []scanRun         `json:"reference"`
	Actual    []scanRun         `json:"actual"`
	Inputs    *diff.ImageInputs `json:"inputs,omitempty"`
	InputLine *scanInputLine    `json:"inputLine,omitempty"`
}

type scanBandOutput struct {
	Query     string            `json:"query"`
	Axis      string            `json:"axis"`
	Start     int               `json:"start"`
	End       int               `json:"end"`
	Length    int               `json:"length"`
	Total     int               `json:"total"`
	Returned  int               `json:"returned"`
	Truncated bool              `json:"truncated"`
	Hint      string            `json:"hint,omitempty"`
	Reference []scanBandLine    `json:"reference"`
	Actual    []scanBandLine    `json:"actual"`
	Inputs    *diff.ImageInputs `json:"inputs,omitempty"`
}

type scanBandLine struct {
	Index int       `json:"index"`
	Runs  []scanRun `json:"runs"`
}

type scanBandSpec struct {
	Axis        string
	Start       int
	End         int
	Flag        string
	Coordinates string
}

type scanInputLine struct {
	Reference scanLinePosition `json:"reference"`
	Actual    scanLinePosition `json:"actual"`
}

type scanLinePosition struct {
	Axis  string `json:"axis"`
	Index int    `json:"index"`
}

type scanRun struct {
	Start  int      `json:"start"`
	End    int      `json:"end"`
	Length int      `json:"length"`
	RGBA   [4]uint8 `json:"rgba"`
	Hex    string   `json:"hex"`
}

func newScanCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "scan <reference.png> <actual.png>",
		Short: "Inspect compact color runs along rows or columns in two PNGs",
		Args:  requireImagePair("scan", "pxp scan reference.png actual.png --row 24"),
		Example: `  pxp scan reference.png actual.png --row 24
  pxp scan reference.png actual.png --rows 20:30
  pxp scan reference.png actual.png --columns 40:48 --format json`,
		RunE: runScanCommand,
	}
	command.Flags().Int("x", 0, "scan vertical column at x in comparison/cropped coordinates")
	command.Flags().Int("y", 0, "scan horizontal row at y in comparison/cropped coordinates")
	command.Flags().Int("column", 0, "alias for --x")
	command.Flags().Int("row", 0, "alias for --y")
	command.Flags().String(
		"rows", "", "scan inclusive row range START:END in comparison/cropped coordinates",
	)
	command.Flags().String(
		"columns", "", "scan inclusive column range START:END in comparison/cropped coordinates",
	)
	addInspectionLimitFlags(command, "runs per image")
	addTabularFormatFlag(command)
	addInputPreparationFlags(command)
	return command
}

func runScanCommand(cmd *cobra.Command, args []string) error {
	limit, err := inspectionResultLimit(cmd)
	if err != nil {
		return cli.NewUsageError(err)
	}
	band, axis, index, err := parseScanSelection(cmd)
	if err != nil {
		return cli.NewUsageError(err)
	}
	format, err := tabularFormat(cmd)
	if err != nil {
		return cli.NewUsageError(err)
	}
	inputs, err := prepareCommandImageInputs(cmd, args[0], args[1])
	if err != nil {
		return err
	}
	defer inputs.cleanup()
	if band != nil {
		output, err := scanBandImages(inputs.referencePath, inputs.actualPath, *band)
		if err != nil {
			return err
		}
		output.Inputs = inputs.metadata
		output.Total = len(output.Reference) + len(output.Actual)
		if !limit.full {
			if len(output.Reference) > limit.maximum {
				output.Reference = output.Reference[:limit.maximum]
			}
			if len(output.Actual) > limit.maximum {
				output.Actual = output.Actual[:limit.maximum]
			}
		}
		output.Returned = len(output.Reference) + len(output.Actual)
		output.Truncated = output.Returned < output.Total
		if output.Truncated {
			output.Hint = cli.FullHint(cmd, args)
		}
		if format == outputpkg.FormatJSON {
			return writeJSON(cmd, output)
		}
		return writeScanBandCSV(cmd, output)
	}

	output, err := scanImages(inputs.referencePath, inputs.actualPath, axis, index)
	if err != nil {
		return err
	}
	output.Inputs = inputs.metadata
	output.InputLine = inputLine(axis, index, inputs.metadata)
	output.Total = len(output.Reference) + len(output.Actual)
	if !limit.full {
		if len(output.Reference) > limit.maximum {
			output.Reference = output.Reference[:limit.maximum]
		}
		if len(output.Actual) > limit.maximum {
			output.Actual = output.Actual[:limit.maximum]
		}
	}
	output.Returned = len(output.Reference) + len(output.Actual)
	output.Truncated = output.Returned < output.Total
	if output.Truncated {
		output.Hint = cli.FullHint(cmd, args)
	}
	if format == outputpkg.FormatJSON {
		return writeJSON(cmd, output)
	}
	return writeScanCSV(cmd, output)
}

func parseScanSelection(command *cobra.Command) (*scanBandSpec, string, int, error) {
	xChanged := command.Flags().Changed("x") || command.Flags().Changed("column")
	yChanged := command.Flags().Changed("y") || command.Flags().Changed("row")
	rowsChanged := command.Flags().Changed("rows")
	columnsChanged := command.Flags().Changed("columns")
	selected := 0
	for _, changed := range []bool{xChanged, yChanged, rowsChanged, columnsChanged} {
		if changed {
			selected++
		}
	}
	if selected != 1 {
		return nil, "", 0, fmt.Errorf(
			"provide exactly one of --x/--column or --y/--row, --rows, or --columns\n\n" +
				"Example: pxp scan reference.png actual.png --rows 20:30",
		)
	}
	if rowsChanged || columnsChanged {
		flag := "--rows"
		axis := "x"
		value, _ := command.Flags().GetString("rows")
		if columnsChanged {
			flag = "--columns"
			axis = "y"
			value, _ = command.Flags().GetString("columns")
		}
		start, end, err := parseScanRange(flag, value)
		if err != nil {
			return nil, "", 0, err
		}
		return &scanBandSpec{
			Axis: axis, Start: start, End: end, Flag: flag, Coordinates: value,
		}, "", 0, nil
	}

	axis := "y"
	index, _ := command.Flags().GetInt("x")
	if command.Flags().Changed("column") {
		index, _ = command.Flags().GetInt("column")
	}
	if yChanged {
		axis = "x"
		index, _ = command.Flags().GetInt("y")
		if command.Flags().Changed("row") {
			index, _ = command.Flags().GetInt("row")
		}
	}
	if index < 0 {
		return nil, "", 0, fmt.Errorf(
			"--%s must be non-negative",
			map[string]string{"x": "y", "y": "x"}[axis],
		)
	}
	return nil, axis, index, nil
}

func parseScanRange(flag, value string) (int, int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, 0, fmt.Errorf(
			"%s must use an inclusive START:END range (for example %s 20:30)", flag, flag,
		)
	}
	start, startErr := strconv.Atoi(parts[0])
	end, endErr := strconv.Atoi(parts[1])
	if startErr != nil || endErr != nil {
		return 0, 0, fmt.Errorf(
			"%s must use integer coordinates in inclusive START:END form (for example %s 20:30)", flag, flag,
		)
	}
	if start < 0 || end < 0 {
		return 0, 0, fmt.Errorf("%s coordinates must be non-negative", flag)
	}
	if start > end {
		return 0, 0, fmt.Errorf("%s start %d must not be greater than end %d", flag, start, end)
	}
	return start, end, nil
}

func inputLine(axis string, index int, inputs *diff.ImageInputs) *scanInputLine {
	if inputs == nil {
		return nil
	}
	return &scanInputLine{
		Reference: lineWithCropOrigin(axis, index, inputs.Reference.Crop),
		Actual:    lineWithCropOrigin(axis, index, inputs.Actual.Crop),
	}
}

func lineWithCropOrigin(axis string, index int, crop *diff.Bounds) scanLinePosition {
	position := scanLinePosition{Axis: axis, Index: index}
	if crop == nil {
		return position
	}
	if axis == "x" {
		position.Index += crop.Y
		return position
	}
	position.Index += crop.X
	return position
}

func scanImages(referencePath, actualPath string, axis string, index int) (scanOutput, error) {
	images, err := imageio.LoadDecodedImages(referencePath, actualPath)
	if err != nil {
		return scanOutput{}, err
	}
	scanAxis := diff.ScanHorizontal
	if axis == "y" {
		scanAxis = diff.ScanVertical
	}
	referenceRuns, actualRuns, err := images.Scan(scanAxis, index)
	if err != nil {
		if boundsErr, ok := err.(*diff.MeasurementBoundsError); ok {
			flag := map[string]string{"x": "--y", "y": "--x"}[axis]
			limit := boundsErr.Size.Y
			if axis == "y" {
				limit = boundsErr.Size.X
			}
			return scanOutput{}, cli.NewUsageError(
				fmt.Errorf(
					"%s index %d is outside image bounds %dx%d (valid 0-%d)",
					flag,
					index,
					boundsErr.Size.X,
					boundsErr.Size.Y,
					limit-1,
				),
			)
		}
		return scanOutput{}, err
	}
	length := boundsLength(images.Reference.Bounds(), scanAxis)
	return scanOutput{
		Axis:      axis,
		Index:     index,
		Length:    length,
		Reference: scanRunsFromDiff(referenceRuns),
		Actual:    scanRunsFromDiff(actualRuns),
	}, nil
}

func scanBandImages(referencePath, actualPath string, spec scanBandSpec) (scanBandOutput, error) {
	images, err := imageio.LoadDecodedImages(referencePath, actualPath)
	if err != nil {
		return scanBandOutput{}, err
	}
	scanAxis := diff.ScanHorizontal
	if spec.Axis == "y" {
		scanAxis = diff.ScanVertical
	}
	referenceLines, actualLines, err := images.ScanBand(scanAxis, spec.Start, spec.End)
	if err != nil {
		if boundsErr, ok := err.(*diff.MeasurementBoundsError); ok {
			limit, _ := scanDimensionsForCommand(boundsErr.Size, spec.Axis)
			return scanBandOutput{}, cli.NewUsageError(
				fmt.Errorf(
					"%s range %s is outside image bounds %dx%d (valid 0:%d, inclusive)",
					spec.Flag,
					spec.Coordinates,
					boundsErr.Size.X,
					boundsErr.Size.Y,
					limit-1,
				),
			)
		}
		return scanBandOutput{}, err
	}
	return scanBandOutput{
		Query:     "scan-band",
		Axis:      spec.Axis,
		Start:     spec.Start,
		End:       spec.End,
		Length:    boundsLength(images.Reference.Bounds(), scanAxis),
		Reference: scanBandLinesFromDiff(referenceLines),
		Actual:    scanBandLinesFromDiff(actualLines),
	}, nil
}

func scanDimensionsForCommand(bounds image.Point, axis string) (int, int) {
	if axis == "y" {
		return bounds.X, bounds.Y
	}
	return bounds.Y, bounds.X
}

func boundsLength(bounds image.Rectangle, axis diff.ScanAxis) int {
	if axis == diff.ScanVertical {
		return bounds.Dy()
	}
	return bounds.Dx()
}

func scanRunsFromDiff(runs []diff.ColorRun) []scanRun {
	output := make([]scanRun, len(runs))
	for index, run := range runs {
		output[index] = scanRun{
			Start:  run.Start,
			End:    run.End,
			Length: run.Length,
			RGBA:   run.RGBA,
			Hex:    formatColorHex(run.RGBA),
		}
	}
	return output
}

func scanBandLinesFromDiff(lines []diff.ScanLine) []scanBandLine {
	output := make([]scanBandLine, len(lines))
	for index, line := range lines {
		output[index] = scanBandLine{Index: line.Index, Runs: scanRunsFromDiff(line.Runs)}
	}
	return output
}

func formatColorHex(rgba [4]uint8) string {
	return fmt.Sprintf("#%02X%02X%02X", rgba[0], rgba[1], rgba[2])
}
