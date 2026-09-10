package main

import (
	"fmt"
	"os"

	"github.com/cristianoliveira/pxp/internal/cli"
	"github.com/cristianoliveira/pxp/internal/commands"
)

func main() {
	command := commands.NewCommand()
	command.SilenceErrors = true
	command.SilenceUsage = true
	if err := command.Execute(); err != nil {
		if renderErr := cli.RenderError(command, err); renderErr != nil {
			fmt.Fprintln(os.Stderr, "failed to render structured error")
		}
		os.Exit(cli.ExitCode(err))
	}
}
