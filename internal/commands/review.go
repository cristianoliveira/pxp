package commands

import (
	"fmt"
	"math"

	"github.com/cristianoliveira/pxp/internal/cli"
	"github.com/cristianoliveira/pxp/internal/review"
	"github.com/spf13/cobra"
)

func newReviewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "review <reference.png> <actual.png>",
		Short: "Review a comparison on localhost and save annotated feedback",
		Args:  requireImagePair("review", "pxp review reference.png actual.png"),
		Example: `  pxp review reference.png actual.png
  pxp review reference.png actual.png --out .pxp-review --previous-feedback .pxp-review/round-123/feedback.json`,
		RunE: runReviewCommand,
	}
	command.Flags().String("out", "", "directory for immutable review round artifacts; defaults to a temporary directory")
	command.Flags().String("previous-feedback", "", "feedback JSON from the preceding round to link without overwriting")
	command.Flags().Uint8("threshold", 0, "ignore per-channel differences at or below this value (0-255)")
	command.Flags().Float64("perceptual-threshold", 0.1, "OKLab HyAB distance above which a pixel is perceptually changed (non-negative)")
	return command
}

func runReviewCommand(cmd *cobra.Command, args []string) error {
	perceptualThreshold, _ := cmd.Flags().GetFloat64("perceptual-threshold")
	if perceptualThreshold < 0 || math.IsNaN(perceptualThreshold) || math.IsInf(perceptualThreshold, 0) {
		return cli.NewUsageError(fmt.Errorf("--perceptual-threshold must be a finite non-negative number"))
	}
	out, _ := cmd.Flags().GetString("out")
	previous, _ := cmd.Flags().GetString("previous-feedback")
	threshold, _ := cmd.Flags().GetUint8("threshold")
	session, err := review.NewSession(args[0], args[1], out, previous, threshold, perceptualThreshold)
	if err != nil {
		return err
	}
	server := review.NewServer(session)
	url, err := server.Start()
	if err != nil {
		return err
	}
	defer server.Close()
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "pxp review listening at %s (Submit feedback or Approve in the browser)\n", url); err != nil {
		return err
	}
	result := server.Wait()
	return writeStructured(cmd, result)
}
