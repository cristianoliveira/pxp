package cli

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// ErrorOutput is the stable process-level failure envelope.
type ErrorOutput struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Category string `json:"category"`
	Message  string `json:"message"`
	Input    string `json:"input,omitempty"`
	ExitCode int    `json:"exitCode"`
	Recovery string `json:"recovery,omitempty"`
}

// ResultError marks a failure whose structured result was already emitted.
type ResultError struct {
	err error
}

func NewResultError(err error) error {
	if err == nil {
		return nil
	}
	return &ResultError{err: err}
}

func (e *ResultError) Error() string { return e.err.Error() }
func (e *ResultError) Unwrap() error { return e.err }

// ErrorContract translates typed command failures into safe domain language.
// The bool is false for intentional silent exits and result-bearing failures.
func ErrorContract(err error) (ErrorOutput, bool) {
	if err == nil || isAlreadyRepresented(err) {
		return ErrorOutput{}, false
	}

	exitCode := ExitCode(err)
	if exitCode == 2 {
		return usageErrorContract(err), true
	}
	return operationalErrorContract(err), true
}

func RenderError(command *cobra.Command, err error) error {
	var result *ResultError
	if errors.As(err, &result) {
		_, writeErr := fmt.Fprintln(command.ErrOrStderr(), result.err.Error())
		return writeErr
	}
	contract, render := ErrorContract(err)
	if !render {
		return nil
	}
	return NewPrinter(command).Structured(contract)
}

func isAlreadyRepresented(err error) bool {
	var silent *ExitCodeError
	if errors.As(err, &silent) {
		return true
	}
	var result *ResultError
	return errors.As(err, &result)
}

func usageErrorContract(err error) ErrorOutput {
	message := err.Error()
	recovery := ""
	var usage *UsageError
	input := ""
	if errors.As(err, &usage) {
		message = usage.Message()
		input = usage.Input()
		recovery = usage.Recovery()
	}
	if strings.HasPrefix(err.Error(), "unknown command ") {
		recovery = "Run the command with --help to list valid commands."
	}
	output := newErrorOutput("usage", message, 2, recovery)
	output.Error.Input = input
	return output
}

func operationalErrorContract(err error) ErrorOutput {
	var path *os.PathError
	if errors.As(err, &path) {
		return newErrorOutput(
			"operational",
			"Could not access a required file.",
			1,
			"Check the file path and permissions, then retry.",
		)
	}

	var pngFormat png.FormatError
	if errors.As(err, &pngFormat) {
		return newErrorOutput(
			"operational",
			"Could not decode an input image.",
			1,
			"Provide valid PNG files and retry.",
		)
	}

	return newErrorOutput(
		"operational",
		"Command could not complete.",
		1,
		"Check inputs and dependencies, then retry.",
	)
}

func newErrorOutput(category, message string, exitCode int, recovery string) ErrorOutput {
	return ErrorOutput{Error: ErrorDetail{
		Category: category,
		Message:  message,
		ExitCode: exitCode,
		Recovery: recovery,
	}}
}
