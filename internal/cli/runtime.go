// Package cli holds shared command-line runtime helpers — error handling and
// dependency wiring used across every cobra command in cmd/. Keeping them here
// lets cmd/ contain only command definitions.
package cli

import (
	"errors"
	"strings"

	"github.com/cristianoliveira/pxp/internal/output"
	"github.com/spf13/cobra"
)

// NewPrinter builds an output.Printer bound to stdout. Structured output is
// TOON by default; the retained global --json flag selects compatibility JSON.
func NewPrinter(cmd *cobra.Command) *output.Printer {
	format := output.FormatTOON
	if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
		format = output.FormatJSON
	}
	return output.New(cmd.OutOrStdout(), format)
}

// ExitCodeError carries a process exit code without a diagnostic message.
// Commands return it to request a silent non-zero exit (for example
// `diff text --quiet` exits 1 when no changes match). Returning it instead of
// calling os.Exit keeps the behaviour testable in-process.
type ExitCodeError struct {
	Code int
}

func (e *ExitCodeError) Error() string { return "" }

// UsageError marks invalid command syntax or arguments. Process entrypoints map
// it to exit code 2 while preserving the original actionable diagnostic.
type UsageError struct {
	err      error
	input    string
	recovery string
}

func NewUsageError(err error) error {
	return NewUsageErrorWithRecovery(err, "")
}

func NewUsageErrorWithRecovery(err error, recovery string) error {
	return NewUsageErrorWithDetails(err, "", recovery)
}

func NewUsageErrorWithDetails(err error, input, recovery string) error {
	if err == nil {
		return nil
	}
	return &UsageError{err: err, input: input, recovery: recovery}
}

func (e *UsageError) Error() string {
	if e.recovery == "" {
		return e.err.Error()
	}
	return e.err.Error() + "\n\n" + e.recovery
}
func (e *UsageError) Unwrap() error    { return e.err }
func (e *UsageError) Message() string  { return e.err.Error() }
func (e *UsageError) Input() string    { return e.input }
func (e *UsageError) Recovery() string { return e.recovery }

// ExitCode maps command errors to the CLI process contract.
func ExitCode(err error) int {
	var explicit *ExitCodeError
	if errors.As(err, &explicit) {
		return explicit.Code
	}
	var usage *UsageError
	if errors.As(err, &usage) {
		return 2
	}
	if err != nil && strings.HasPrefix(err.Error(), "unknown command ") {
		return 2
	}
	return 1
}

// MarkUsageErrors classifies Cobra's positional-argument failures.
// Flag handlers must mark their errors while preserving command-specific help.
// RunE errors remain operational unless command code explicitly marks them.
func MarkUsageErrors(command *cobra.Command) {
	if command.Args != nil {
		validateArgs := command.Args
		command.Args = func(cmd *cobra.Command, args []string) error {
			return NewUsageErrorWithRecovery(
				validateArgs(cmd, args),
				"Run `"+cmd.CommandPath()+" --help` for valid usage.",
			)
		}
	}
	for _, child := range command.Commands() {
		MarkUsageErrors(child)
	}
}
