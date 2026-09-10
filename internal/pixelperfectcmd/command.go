package pixelperfectcmd

import (
	"fmt"

	"github.com/cristianoliveira/pxp/internal/cli"
	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	outputpkg "github.com/cristianoliveira/pxp/internal/output"
	"github.com/spf13/cobra"
)

func newCommand(compare imageComparer) *cobra.Command {
	command := &cobra.Command{
		Use:   "image <reference.png> <actual.png>",
		Short: "Compare equal-sized PNGs and write a changed-pixel mask",
		Args:  requireImagePair("compare", "pxp reference.png actual.png"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runComparisonCommand(cmd, args, compare)
		},
	}
	command.PersistentFlags().Bool("json", false, "emit structured results as compatibility JSON")
	command.Flags().String("profile", "", "load comparison options from a versioned JSON profile; explicit flags override profile values")
	command.Flags().String("annotations", "", "enrich mismatch regions from a generic coordinate annotation JSON file")
	command.Flags().StringP("output", "o", "", "path for transparent PNG difference mask; defaults to <actual>.diff.png")
	command.Flags().Uint8("threshold", 0, "ignore per-channel differences at or below this value (0-255)")
	command.Flags().Float64("perceptual-threshold", diff.DefaultPerceptualThreshold, "OKLab HyAB distance above which a pixel is perceptually changed (non-negative)")
	command.Flags().String("region", "", "compare only x,y,width,height")
	command.Flags().String("reference-crop", "", "crop reference before comparing: x,y,width,height")
	command.Flags().String("reference-metadata", "", "apply logical crop from image export metadata JSON")
	command.Flags().String("actual-crop", "", "crop actual before comparing: x,y,width,height")
	command.Flags().StringArray("ignore-region", nil, "exclude x,y,width,height; repeat for multiple areas")
	command.Flags().String("mask", "", "full-size PNG selecting compared pixels (visible non-black includes)")
	command.Flags().String("overlay", "", "path for directional overlay (reference red, actual green)")
	command.Flags().String("report", "", "write a self-contained HTML report to this path")
	command.Flags().Int("suggest-offset", 0, "report best whole-image translation within this pixel radius without applying it")
	command.Flags().Int("suggest-movement", 0, "report advisory per-region translations within this pixel radius without applying them")
	command.Flags().Int("region-gap", 0, "group mismatch regions separated by at most this many pixels")
	command.Flags().Int("min-region-pixels", 1, "omit disconnected regions smaller than this many changed pixels")
	command.Flags().Int("max-regions", 20, "maximum mismatch regions included in output")
	command.Flags().Bool("full", false, "include every mismatch region")
	command.Flags().Float64("max-rmse", -1, "fail when normalized RMSE exceeds this value")
	command.Flags().Float64("max-changed-ratio", -1, "fail when changed-pixel ratio exceeds this value")
	command.Flags().Float64("max-perceptual-changed-ratio", -1, "fail when perceptual changed-pixel ratio exceeds this value")
	command.Flags().Bool("visual-context", false, "add advisory visual descriptions using the configured multimodal model")
	command.Flags().String("visual-context-provider", "openrouter", "visual context provider: openrouter or openai")
	command.Flags().String("visual-context-model", "", "override the visual context model")
	command.Flags().String("visual-context-prompt", "", "extra advisory focus for visual context analysis")
	command.AddCommand(newProbeCommand())
	command.AddCommand(newScanCommand())
	return command
}

func addTabularFormatFlag(command *cobra.Command) {
	command.Flags().String("format", string(outputpkg.FormatCSV), "output format: csv or json")
}

func tabularFormat(command *cobra.Command) (outputpkg.Format, error) {
	value, _ := command.Flags().GetString("format")
	return outputpkg.ParseFormat(value, outputpkg.FormatCSV, outputpkg.FormatJSON)
}

const defaultInspectionLimit = 25

type inspectionLimit struct {
	maximum int
	full    bool
}

func addInspectionLimitFlags(command *cobra.Command, noun string) {
	command.Flags().Int("limit", defaultInspectionLimit, "maximum "+noun+" to emit")
	command.Flags().Bool("full", false, "emit all "+noun)
}

func inspectionResultLimit(command *cobra.Command) (inspectionLimit, error) {
	maximum, _ := command.Flags().GetInt("limit")
	full, _ := command.Flags().GetBool("full")
	if maximum <= 0 {
		return inspectionLimit{}, fmt.Errorf("--limit must be greater than zero")
	}
	if full && command.Flags().Changed("limit") {
		return inspectionLimit{}, fmt.Errorf("--full cannot be combined with --limit")
	}
	return inspectionLimit{maximum: maximum, full: full}, nil
}

func NewCommand() *cobra.Command {
	return newCommandWithExecutable(cli.CurrentExecutablePath)
}

func newCommandWithExecutable(resolveExecutable func() (string, error)) *cobra.Command {
	command := newCommand(nil)
	command.Use = "pxp <reference.png> <actual.png>"
	command.Example = `  pxp reference.png actual.png
  pxp reference.png actual.png --overlay overlay.png
  pxp reference.png actual.png --max-changed-ratio 0.01`
	validateArgs := command.Args
	command.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && cmd.Flags().NFlag() == 0 {
			return nil
		}
		return validateArgs(cmd, args)
	}
	run := command.RunE
	command.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return run(cmd, args)
		}
		executable, err := resolveExecutable()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), `pxp harnesses image comparison.
Executable: %s
Usage: pxp <reference.png> <actual.png>
Next:
  pxp reference.png actual.png
  pxp probe --help
  pxp scan --help
`, executable)
		return err
	}
	command.SetFlagErrorFunc(cli.NewFlagUsageError)
	cli.MarkUsageErrors(command)
	return command
}

func requireImagePair(action, example string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 2 {
			return nil
		}
		return fmt.Errorf("%s requires <reference.png> and <actual.png>; received %d argument(s)\n\nExample: %s", action, len(args), example)
	}
}
