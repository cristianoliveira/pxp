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
  pxp review reference.png actual.png --out .pxp-review \
    --previous-feedback .pxp-review/round-123/feedback.json`,
		RunE: runReviewCommand,
	}
	command.Flags().String(
		"out",
		"",
		"directory for immutable review round artifacts; "+
			"defaults to a temporary directory",
	)
	command.Flags().String(
		"previous-feedback",
		"",
		"feedback JSON from the preceding round to link without overwriting",
	)
	command.Flags().Uint8(
		"threshold",
		0,
		"ignore per-channel differences at or below this value (0-255)",
	)
	command.Flags().Float64(
		"perceptual-threshold",
		0.1,
		"OKLab HyAB distance above which a pixel is perceptually changed "+
			"(non-negative)",
	)
	return command
}

type reviewOptions struct {
	outputRoot          string
	previousFeedback    string
	threshold           uint8
	perceptualThreshold float64
}

func reviewOptionsFrom(command *cobra.Command) (reviewOptions, error) {
	perceptualThreshold, _ := command.Flags().GetFloat64("perceptual-threshold")
	if perceptualThreshold < 0 || math.IsNaN(perceptualThreshold) ||
		math.IsInf(perceptualThreshold, 0) {
		return reviewOptions{}, cli.NewUsageError(
			fmt.Errorf("--perceptual-threshold must be a finite non-negative number"),
		)
	}
	outputRoot, _ := command.Flags().GetString("out")
	previousFeedback, _ := command.Flags().GetString("previous-feedback")
	threshold, _ := command.Flags().GetUint8("threshold")
	return reviewOptions{
		outputRoot:          outputRoot,
		previousFeedback:    previousFeedback,
		threshold:           threshold,
		perceptualThreshold: perceptualThreshold,
	}, nil
}

func runReviewCommand(cmd *cobra.Command, args []string) error {
	options, err := reviewOptionsFrom(cmd)
	if err != nil {
		return err
	}
	session, err := review.NewSession(
		args[0],
		args[1],
		options.outputRoot,
		options.previousFeedback,
		options.threshold,
		options.perceptualThreshold,
	)
	if err != nil {
		return err
	}
	server := review.NewServer(session)
	url, err := server.Start()
	if err != nil {
		return err
	}
	defer func() { _ = server.Close() }()
	message := fmt.Sprintf(
		"pxp review listening at %s (Submit feedback or Approve in the browser)",
		url,
	)
	if _, err := fmt.Fprintln(cmd.ErrOrStderr(), message); err != nil {
		return err
	}
	result, err := server.WaitContext(cmd.Context())
	if err != nil {
		return err
	}
	return writeStructured(cmd, result)
}
