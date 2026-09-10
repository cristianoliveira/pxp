package commands

import (
	"fmt"
	"image"

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
		Short: "Inspect compact color runs along one row or column in two PNGs",
		Args:  requireImagePair("scan", "pxp scan reference.png actual.png --row 24"),
		Example: `  pxp scan reference.png actual.png --row 24
  pxp scan reference.png actual.png --column 12 --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, err := inspectionResultLimit(cmd)
			if err != nil {
				return cli.NewUsageError(err)
			}
			xChanged := cmd.Flags().Changed("x") || cmd.Flags().Changed("column")
			yChanged := cmd.Flags().Changed("y") || cmd.Flags().Changed("row")
			if xChanged == yChanged {
				return cli.NewUsageError(fmt.Errorf("provide exactly one of --x/--column or --y/--row"))
			}
			axis := "y"
			index, _ := cmd.Flags().GetInt("x")
			if cmd.Flags().Changed("column") {
				index, _ = cmd.Flags().GetInt("column")
			}
			if yChanged {
				axis = "x"
				index, _ = cmd.Flags().GetInt("y")
				if cmd.Flags().Changed("row") {
					index, _ = cmd.Flags().GetInt("row")
				}
			}
			if index < 0 {
				return cli.NewUsageError(fmt.Errorf("--%s must be non-negative", map[string]string{"x": "y", "y": "x"}[axis]))
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
		},
	}
	command.Flags().Int("x", 0, "scan vertical column at x in comparison/cropped coordinates")
	command.Flags().Int("y", 0, "scan horizontal row at y in comparison/cropped coordinates")
	command.Flags().Int("column", 0, "alias for --x")
	command.Flags().Int("row", 0, "alias for --y")
	addInspectionLimitFlags(command, "runs per image")
	addTabularFormatFlag(command)
	addInputPreparationFlags(command)
	return command
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
			return scanOutput{}, cli.NewUsageError(fmt.Errorf("%s index %d is outside image bounds %dx%d (valid 0-%d)", flag, index, boundsErr.Size.X, boundsErr.Size.Y, limit-1))
		}
		return scanOutput{}, err
	}
	length := boundsLength(images.Reference.Bounds(), scanAxis)
	return scanOutput{Axis: axis, Index: index, Length: length, Reference: scanRunsFromDiff(referenceRuns), Actual: scanRunsFromDiff(actualRuns)}, nil
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
		output[index] = scanRun{Start: run.Start, End: run.End, Length: run.Length, RGBA: run.RGBA, Hex: formatColorHex(run.RGBA)}
	}
	return output
}

func formatColorHex(rgba [4]uint8) string {
	return fmt.Sprintf("#%02X%02X%02X", rgba[0], rgba[1], rgba[2])
}
