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
	command.Flags().String(
		"context-file",
		"",
		"JSON file with implementation context for this review round",
	)
	command.Flags().Bool(
		"open",
		false,
		"open the review URL in the default browser and wait for completion",
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
	contextFile         string
	context             review.ImplementationContext
	threshold           uint8
	perceptualThreshold float64
	openBrowser         bool
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
	contextFile, _ := command.Flags().GetString("context-file")
	context, err := review.LoadContext(contextFile)
	if err != nil {
		return reviewOptions{}, cli.NewUsageError(err)
	}
	threshold, _ := command.Flags().GetUint8("threshold")
	openBrowser, _ := command.Flags().GetBool("open")
	return reviewOptions{
		outputRoot:          outputRoot,
		previousFeedback:    previousFeedback,
		contextFile:         contextFile,
		context:             context,
		threshold:           threshold,
		perceptualThreshold: perceptualThreshold,
		openBrowser:         openBrowser,
	}, nil
}

func runReviewCommand(cmd *cobra.Command, args []string) error {
	return runReviewCommandWithBrowser(cmd, args, openDefaultBrowser)
}

func runReviewCommandWithBrowser(cmd *cobra.Command, args []string, open browserOpener) error {
	options, err := reviewOptionsFrom(cmd)
	if err != nil {
		return err
	}
	session, err := review.NewSessionWithContext(
		args[0],
		args[1],
		options.outputRoot,
		options.previousFeedback,
		options.threshold,
		options.perceptualThreshold,
		options.context,
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
	if options.openBrowser {
		if err := open(cmd.Context(), url); err != nil {
			if _, writeErr := fmt.Fprintf(
				cmd.ErrOrStderr(),
				"pxp review browser open failed: %v; open %s manually\n",
				err,
				url,
			); writeErr != nil {
				return writeErr
			}
		}
	}
	result, err := server.WaitContext(cmd.Context())
	if err != nil {
		return err
	}
	return writeStructured(cmd, result)
}
