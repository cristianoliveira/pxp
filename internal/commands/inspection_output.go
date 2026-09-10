package commands

import (
	"fmt"
	"io"
	"strconv"

	"github.com/cristianoliveira/pxp/internal/cli"
	outputpkg "github.com/cristianoliveira/pxp/internal/output"
	"github.com/spf13/cobra"
)

func writeProbeCSV(command *cobra.Command, output probeOutput) error {
	writer := command.OutOrStdout()
	if _, err := fmt.Fprintln(writer, "x,y,ref,act,delta,input_ref,input_act"); err != nil {
		return err
	}
	for _, point := range output.Points {
		inputReference, inputActual := "", ""
		if point.InputPoint != nil {
			inputReference = formatProbePoint(point.InputPoint.Reference)
			inputActual = formatProbePoint(point.InputPoint.Actual)
		}
		if _, err := fmt.Fprintf(writer, "%d,%d,%s,%s,%d,%s,%s\n", point.Point.X, point.Point.Y, point.Reference.Hex, point.Actual.Hex, maxAbsDelta(point.Delta), inputReference, inputActual); err != nil {
			return err
		}
	}
	return writeTruncationCSV(writer, output.Total, output.Returned, output.Truncated, output.Hint)
}

func writeScanCSV(command *cobra.Command, output scanOutput) error {
	writer := command.OutOrStdout()
	if _, err := fmt.Fprintln(writer, "image,axis,index,start,end,length,hex,input_axis,input_index"); err != nil {
		return err
	}
	if err := writeScanRunsCSV(writer, "ref", output.Axis, output.Index, output.Reference, output.InputLine, true); err != nil {
		return err
	}
	if err := writeScanRunsCSV(writer, "act", output.Axis, output.Index, output.Actual, output.InputLine, false); err != nil {
		return err
	}
	return writeTruncationCSV(writer, output.Total, output.Returned, output.Truncated, output.Hint)
}

func writeTruncationCSV(writer io.Writer, total, returned int, truncated bool, hint string) error {
	if !truncated {
		return nil
	}
	_, err := fmt.Fprintf(writer, "# total=%d returned=%d truncated=true hint=%q\n", total, returned, hint)
	return err
}

func writeScanRunsCSV(writer io.Writer, imageName string, axis string, index int, runs []scanRun, inputLine *scanInputLine, reference bool) error {
	inputAxis, inputIndex := "", ""
	if inputLine != nil {
		line := inputLine.Actual
		if reference {
			line = inputLine.Reference
		}
		inputAxis = line.Axis
		inputIndex = strconv.Itoa(line.Index)
	}
	for _, run := range runs {
		if _, err := fmt.Fprintf(writer, "%s,%s,%d,%d,%d,%d,%s,%s,%s\n", imageName, axis, index, run.Start, run.End, run.Length, run.Hex, inputAxis, inputIndex); err != nil {
			return err
		}
	}
	return nil
}

func formatProbePoint(point probePoint) string {
	return fmt.Sprintf("%d:%d", point.X, point.Y)
}

func maxAbsDelta(delta probeDelta) int {
	return max(max(absInt(delta.R), absInt(delta.G)), max(absInt(delta.B), absInt(delta.A)))
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func writeStructured(command *cobra.Command, value any) error {
	return cli.NewPrinter(command).Structured(value)
}

func writeJSON(command *cobra.Command, value any) error {
	return outputpkg.New(command.OutOrStdout(), outputpkg.FormatJSON).JSON(value)
}

// NewCommand creates the standalone image comparison command.
