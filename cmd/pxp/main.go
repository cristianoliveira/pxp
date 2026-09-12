package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cristianoliveira/pxp/internal/cli"
	"github.com/cristianoliveira/pxp/internal/commands"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	command := commands.NewCommand()
	command.SetContext(ctx)
	command.SilenceErrors = true
	command.SilenceUsage = true
	if err := command.Execute(); err != nil {
		if renderErr := cli.RenderError(command, err); renderErr != nil {
			fmt.Fprintln(os.Stderr, "failed to render structured error")
		}
		os.Exit(cli.ExitCode(err))
	}
}
