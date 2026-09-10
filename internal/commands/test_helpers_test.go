package commands

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"testing"

	"github.com/spf13/cobra"

	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/stretchr/testify/require"
)

func parseIgnoredRegionsError(values []string) error {
	_, err := parseIgnoredRegions(values)
	return err
}

func extractFirstRegion(t *testing.T, output string) diff.Region {
	t.Helper()
	var comparison diff.ImageComparison
	require.NoError(t, json.Unmarshal([]byte(output), &comparison))
	require.NotEmpty(t, comparison.Regions)
	return comparison.Regions[0]
}

func writeTestPNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(file, img))
	require.NoError(t, file.Close())
}

func executeCommand(command *cobra.Command, args ...string) commandResult {
	var stdout, stderr bytes.Buffer
	command.SilenceErrors = true
	command.SilenceUsage = true
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	if len(args) > 0 {
		args = append([]string{"--json"}, args...)
	}
	command.SetArgs(args)
	err := command.Execute()
	return commandResult{Stdout: stdout.String(), Stderr: stderr.String(), Err: err}
}

type commandResult struct {
	Stdout, Stderr string
	Err            error
}
