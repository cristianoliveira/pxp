package commands

import (
	"encoding/json"
	"image"
	"image/color"
	"path/filepath"
	"testing"

	"github.com/cristianoliveira/pxp/internal/cli"
	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnatomyCommandFindsFixtureElements(t *testing.T) {
	fixture := filepath.Join("..", "..", "skills", "pxp", "fixtures", "card-alert.png")
	result := executeCommand(NewCommand(), "anatomy", fixture, "--full", "--group", "8")

	require.NoError(t, result.Err)
	var output diff.AnatomyResult
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
	assert.Equal(t, 656, output.Width)
	assert.Equal(t, 344, output.Height)
	assert.Equal(t, "#FFFFFF", output.Background)
	assert.Equal(t, 9, output.Total)
	assert.Equal(t, output.Total, output.Returned)
	assert.False(t, output.Truncated)
	assert.Equal(t, diff.Bounds{X: 36, Y: 36, Width: 56, Height: 56}, output.Elements[0].Bounds)
	assert.Equal(t, diff.Bounds{X: 32, Y: 244, Width: 188, Height: 64}, output.Elements[7].Bounds)
}

func TestAnatomyCommandReturnsExplicitEmptyResult(t *testing.T) {
	path := writeSolidAnatomyImage(t)
	result := executeCommand(NewCommand(), "anatomy", path)

	require.NoError(t, result.Err)
	var output diff.AnatomyResult
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
	assert.Equal(t, "foreground-elements", output.Query)
	assert.Contains(t, output.Message, "no foreground-elements found")
	assert.Empty(t, output.Elements)
}

func TestAnatomyCommandTruncationProvidesFullHint(t *testing.T) {
	path := writeSolidAnatomyImage(t)
	// Add two separated foreground blocks to make the limit observable.
	img := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	for y := 1; y < 3; y++ {
		for x := 1; x < 3; x++ {
			img.SetNRGBA(x, y, color.NRGBA{A: 255})
			img.SetNRGBA(x+8, y, color.NRGBA{A: 255})
		}
	}
	writeTestPNG(t, path, img)

	result := executeCommand(NewCommand(), "anatomy", path, "--limit", "1")

	require.NoError(t, result.Err)
	var output diff.AnatomyResult
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
	assert.Equal(t, 2, output.Total)
	assert.Equal(t, 1, output.Returned)
	assert.True(t, output.Truncated)
	assert.Contains(t, output.Hint, "--full")
}

func TestAnatomyCommandRejectsInvalidBackgroundAndMissingInput(t *testing.T) {
	invalid := executeCommand(NewCommand(), "anatomy", "image.png", "--background", "white")
	assert.Equal(t, 2, cli.ExitCode(invalid.Err))
	assert.ErrorContains(t, invalid.Err, "--background must be #RRGGBB")

	missing := executeCommand(NewCommand(), "anatomy", "missing-image.png")
	assert.NotNil(t, missing.Err)
	assert.ErrorContains(t, missing.Err, "missing-image.png")
}

func writeSolidAnatomyImage(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "image.png")
	writeTestPNG(t, path, image.NewNRGBA(image.Rect(0, 0, 4, 4)))
	return path
}
