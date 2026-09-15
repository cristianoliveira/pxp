package commands

import (
	"encoding/json"
	"image"
	"image/color"
	"path/filepath"
	"testing"

	clipkg "github.com/cristianoliveira/pxp/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanCommandReportsExactRowsAsBandJSON(t *testing.T) {
	directory := t.TempDir()
	referencePath := filepath.Join(directory, "reference.png")
	actualPath := filepath.Join(directory, "actual.png")
	reference := image.NewRGBA(image.Rect(0, 0, 4, 4))
	actual := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			reference.SetRGBA(x, y, color.RGBA{R: uint8(y + 1), A: 255})
			actual.SetRGBA(x, y, color.RGBA{G: uint8(y + 1), A: 255})
		}
	}
	reference.SetRGBA(2, 1, color.RGBA{B: 255, A: 255})
	writeTestPNG(t, referencePath, reference)
	writeTestPNG(t, actualPath, actual)

	result := executeCommand(NewCommand(), "scan", referencePath, actualPath, "--rows", "1:2", "--format", "json")

	require.NoError(t, result.Err)
	var output scanBandOutput
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
	assert.Equal(t, "scan-band", output.Query)
	assert.Equal(t, "x", output.Axis)
	assert.Equal(t, 1, output.Start)
	assert.Equal(t, 2, output.End)
	assert.Equal(t, 4, output.Length)
	assert.Equal(t, 4, output.Total)
	assert.Equal(t, 4, output.Returned)
	assert.False(t, output.Truncated)
	require.Len(t, output.Reference, 2)
	assert.Equal(t, []int{1, 2}, []int{output.Reference[0].Index, output.Reference[1].Index})
	assert.Len(t, output.Reference[0].Runs, 3)
	assert.Equal(t, "#0000FF", output.Reference[0].Runs[1].Hex)
	assert.Equal(t, "#030000", output.Reference[1].Runs[0].Hex)
	assert.Equal(t, "#000300", output.Actual[1].Runs[0].Hex)
}

func TestScanCommandReportsExactColumnsAndCSVTruncation(t *testing.T) {
	directory := t.TempDir()
	referencePath := filepath.Join(directory, "reference.png")
	actualPath := filepath.Join(directory, "actual.png")
	reference := image.NewRGBA(image.Rect(0, 0, 3, 4))
	actual := image.NewRGBA(image.Rect(0, 0, 3, 4))
	for x := 0; x < 3; x++ {
		for y := 0; y < 4; y++ {
			reference.SetRGBA(x, y, color.RGBA{R: uint8(x + 1), A: 255})
			actual.SetRGBA(x, y, color.RGBA{G: uint8(x + 1), A: 255})
		}
	}
	writeTestPNG(t, referencePath, reference)
	writeTestPNG(t, actualPath, actual)

	result := executeCommand(NewCommand(), "scan", referencePath, actualPath, "--columns", "1:2", "--limit", "1")

	require.NoError(t, result.Err)
	assert.Contains(t, result.Stdout, "ref,y,1,0,3,4,#020000")
	assert.Contains(t, result.Stdout, "act,y,1,0,3,4,#000200")
	assert.NotContains(t, result.Stdout, "ref,y,2,")
	assert.Contains(t, result.Stdout, "# total=4 returned=2 truncated=true")
	assert.Contains(t, result.Stdout, "--columns 1:2")
	assert.Contains(t, result.Stdout, "--full")
}

func TestScanCommandBandAppliesCropsAndIsDeterministic(t *testing.T) {
	directory := t.TempDir()
	referencePath := filepath.Join(directory, "reference.png")
	actualPath := filepath.Join(directory, "actual.png")
	reference := image.NewRGBA(image.Rect(0, 0, 4, 4))
	actual := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			reference.SetRGBA(x, y, color.RGBA{R: uint8(x + y), A: 255})
			actual.SetRGBA(x, y, color.RGBA{G: uint8(x + y), A: 255})
		}
	}
	writeTestPNG(t, referencePath, reference)
	writeTestPNG(t, actualPath, actual)
	args := []string{
		"scan", referencePath, actualPath,
		"--reference-crop", "1,1,2,2", "--actual-crop", "1,1,2,2",
		"--rows", "0:1", "--format", "json",
	}

	first := executeCommand(NewCommand(), args...)
	second := executeCommand(NewCommand(), args...)

	require.NoError(t, first.Err)
	require.NoError(t, second.Err)
	assert.Equal(t, first.Stdout, second.Stdout)
	var output scanBandOutput
	require.NoError(t, json.Unmarshal([]byte(first.Stdout), &output))
	assert.Equal(t, 0, output.Start)
	assert.Equal(t, 1, output.End)
	assert.Equal(t, 2, output.Length)
	require.NotNil(t, output.Inputs)
	assert.Equal(t, 1, output.Inputs.Reference.Crop.X)
	assert.Equal(t, 1, output.Inputs.Reference.Crop.Y)
}

func TestScanCommandBandValidatesRangesBeforeReadingImages(t *testing.T) {
	tests := []struct {
		name, expected string
		args           []string
	}{
		{name: "malformed", args: []string{"--rows", "2"}, expected: "--rows must use an inclusive START:END range"},
		{name: "empty", args: []string{"--columns", ":"}, expected: "--columns must use an inclusive START:END range"},
		{name: "reversed", args: []string{"--rows", "3:1"}, expected: "--rows start 3 must not be greater than end 1"},
		{name: "negative", args: []string{"--rows", "-1:2"}, expected: "--rows coordinates must be non-negative"},
		{name: "mixed axis", args: []string{"--rows", "1:2", "--column", "1"}, expected: "provide exactly one of"},
		{name: "invalid format", args: []string{"--rows", "1:2", "--format", "yaml"}, expected: `invalid --format "yaml": expected csv or json`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"scan", "missing-reference.png", "missing-actual.png"}, test.args...)
			result := executeCommand(NewCommand(), args...)
			require.Error(t, result.Err)
			assert.Equal(t, 2, clipkg.ExitCode(result.Err))
			assert.ErrorContains(t, result.Err, test.expected)
			assert.NotContains(t, result.Err.Error(), "decode reference")
		})
	}
}

func TestScanCommandBandRejectsOutOfBoundsRange(t *testing.T) {
	directory := t.TempDir()
	referencePath := filepath.Join(directory, "reference.png")
	actualPath := filepath.Join(directory, "actual.png")
	writeTestPNG(t, referencePath, image.NewRGBA(image.Rect(0, 0, 3, 3)))
	writeTestPNG(t, actualPath, image.NewRGBA(image.Rect(0, 0, 3, 3)))

	result := executeCommand(NewCommand(), "scan", referencePath, actualPath, "--columns", "0:3")

	require.Error(t, result.Err)
	assert.Equal(t, 2, clipkg.ExitCode(result.Err))
	assert.ErrorContains(t, result.Err, "--columns range 0:3 is outside image bounds")
	assert.ErrorContains(t, result.Err, "valid 0:2, inclusive")
}
