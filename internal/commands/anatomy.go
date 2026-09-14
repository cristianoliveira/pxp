package commands

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/cristianoliveira/pxp/internal/cli"
	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/cristianoliveira/pxp/internal/imageio"
	outputpkg "github.com/cristianoliveira/pxp/internal/output"
	"github.com/spf13/cobra"
)

func newAnatomyCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "anatomy <image.png>",
		Short: "Discover deterministic foreground element bounds in one PNG",
		Args:  requireSingleImage("anatomy", "pxp anatomy image.png"),
		Example: `  pxp anatomy screenshot.png
  pxp anatomy screenshot.png --json --full
  pxp anatomy screenshot.png --group 8 --min-pixels 8 --format toon`,
		RunE: runAnatomyCommand,
	}
	command.Flags().Uint8(
		"threshold", 8, "ignore per-channel differences at or below this value (0-255)",
	)
	command.Flags().String("background", "", "override detected background with #RRGGBB")
	command.Flags().Int("group", 0, "merge elements whose bounds are within this pixel gap")
	command.Flags().Int("min-pixels", 4, "ignore elements smaller than this pixel count")
	command.Flags().Int("limit", defaultInspectionLimit, "maximum elements to emit")
	command.Flags().Bool("full", false, "emit every discovered element")
	command.Flags().String("format", string(outputpkg.FormatTOON), "output format: toon or json")
	return command
}

func runAnatomyCommand(command *cobra.Command, args []string) error {
	limit, err := inspectionResultLimit(command)
	if err != nil {
		return cli.NewUsageError(err)
	}
	threshold, _ := command.Flags().GetUint8("threshold")
	group, _ := command.Flags().GetInt("group")
	minPixels, _ := command.Flags().GetInt("min-pixels")
	if group < 0 {
		return cli.NewUsageError(fmt.Errorf("--group must be non-negative"))
	}
	if minPixels < 1 {
		return cli.NewUsageError(fmt.Errorf("--min-pixels must be greater than zero"))
	}
	background, err := anatomyBackground(command)
	if err != nil {
		return cli.NewUsageError(err)
	}
	format, err := anatomyFormat(command)
	if err != nil {
		return cli.NewUsageError(err)
	}
	img, err := imageio.LoadDecodedImage(args[0])
	if err != nil {
		return err
	}
	result, err := diff.DiscoverAnatomy(img, diff.AnatomyOptions{
		Threshold: threshold, Background: background, Group: group, MinPixels: minPixels,
	})
	if err != nil {
		return cli.NewUsageError(err)
	}
	if !limit.full && len(result.Elements) > limit.maximum {
		result.Elements = result.Elements[:limit.maximum]
	}
	result.Returned = len(result.Elements)
	result.Truncated = result.Returned < result.Total
	if result.Truncated {
		result.Hint = cli.FullHint(command, args)
	}
	return outputpkg.New(command.OutOrStdout(), format).Structured(result)
}

func requireSingleImage(action, example string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 1 {
			return nil
		}
		return fmt.Errorf(
			"%s requires <image.png>; received %d argument(s)\n\nExample: %s",
			action, len(args), example,
		)
	}
}

func anatomyBackground(command *cobra.Command) (*[4]uint8, error) {
	value, _ := command.Flags().GetString("background")
	if value == "" {
		return nil, nil
	}
	if !strings.HasPrefix(value, "#") || len(value) != 7 {
		return nil, fmt.Errorf("--background must be #RRGGBB")
	}
	decoded, err := hex.DecodeString(value[1:])
	if err != nil {
		return nil, fmt.Errorf("--background must be #RRGGBB")
	}
	background := &[4]uint8{decoded[0], decoded[1], decoded[2], 255}
	return background, nil
}

func anatomyFormat(command *cobra.Command) (outputpkg.Format, error) {
	asJSON, _ := command.Flags().GetBool("json")
	if asJSON {
		return outputpkg.FormatJSON, nil
	}
	value, _ := command.Flags().GetString("format")
	return outputpkg.ParseFormat(value, outputpkg.FormatTOON, outputpkg.FormatJSON)
}
