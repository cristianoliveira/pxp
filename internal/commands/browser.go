package commands

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

const browserLaunchTimeout = 5 * time.Second

type browserOpener func(context.Context, string) error

func openDefaultBrowser(parent context.Context, url string) error {
	command, args, ok := browserCommand(runtime.GOOS, url)
	if !ok {
		return fmt.Errorf("automatic browser opening is unsupported on %s", runtime.GOOS)
	}

	ctx, cancel := context.WithTimeout(parent, browserLaunchTimeout)
	defer cancel()
	if err := exec.CommandContext(ctx, command, args...).Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("browser launcher timed out: %w", ctx.Err())
		}
		return fmt.Errorf("browser launcher %q failed: %w", command, err)
	}
	return nil
}

func browserCommand(goos, url string) (string, []string, bool) {
	switch goos {
	case "darwin":
		return "open", []string{url}, true
	case "linux":
		return "xdg-open", []string{url}, true
	case "windows":
		return "rundll32.exe", []string{"url.dll,FileProtocolHandler", url}, true
	default:
		return "", nil, false
	}
}
