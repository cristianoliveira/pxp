package pixelperfectcmd

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

func TestScanCommandReportsRowColorRuns(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	referenceImage := image.NewRGBA(image.Rect(0, 0, 4, 1))
	actualImage := image.NewRGBA(image.Rect(0, 0, 4, 1))
	for x := 0; x < 2; x++ {
		referenceImage.SetRGBA(x, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		actualImage.SetRGBA(x, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	}
	for x := 2; x < 4; x++ {
		referenceImage.SetRGBA(x, 0, color.RGBA{R: 222, G: 223, B: 224, A: 255})
		actualImage.SetRGBA(x, 0, color.RGBA{R: 233, G: 235, B: 236, A: 255})
	}
	writeTestPNG(t, reference, referenceImage)
	writeTestPNG(t, actual, actualImage)

	result := executeCommand(NewCommand(), "scan", reference, actual, "--y", "0")

	require.NoError(t, result.Err)
	assert.Equal(t, "image,axis,index,start,end,length,hex,input_axis,input_index\nref,x,0,0,1,2,#FFFFFF,,\nref,x,0,2,3,2,#DEDFE0,,\nact,x,0,0,1,2,#FFFFFF,,\nact,x,0,2,3,2,#E9EBEC,,\n", result.Stdout)
}

func TestScanCommandBoundsRunsAndProvidesScopePreservingHint(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	referenceImage := image.NewRGBA(image.Rect(0, 0, 4, 1))
	actualImage := image.NewRGBA(image.Rect(0, 0, 4, 1))
	for x := 0; x < 4; x++ {
		referenceImage.SetRGBA(x, 0, color.RGBA{R: uint8(x * 20), A: 255})
		actualImage.SetRGBA(x, 0, color.RGBA{G: uint8(x * 20), A: 255})
	}
	writeTestPNG(t, reference, referenceImage)
	writeTestPNG(t, actual, actualImage)

	result := executeCommand(NewCommand(), "scan", reference, actual, "--row", "0", "--limit", "2", "--format", "json")

	require.NoError(t, result.Err)
	var output scanOutput
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
	assert.Equal(t, 8, output.Total)
	assert.Equal(t, 4, output.Returned)
	assert.True(t, output.Truncated)
	assert.Len(t, output.Reference, 2)
	assert.Len(t, output.Actual, 2)
	assert.Contains(t, output.Hint, "--row 0")
	assert.Contains(t, output.Hint, "--format json")
	assert.Contains(t, output.Hint, "--full")

	exact := executeCommand(NewCommand(), "scan", reference, actual, "--row", "0", "--limit", "4", "--format", "json")
	require.NoError(t, exact.Err)
	output = scanOutput{}
	require.NoError(t, json.Unmarshal([]byte(exact.Stdout), &output))
	assert.Equal(t, 8, output.Total)
	assert.Equal(t, 8, output.Returned)
	assert.False(t, output.Truncated)
	assert.Empty(t, output.Hint)
}

func TestScanCommandAppliesInputCrops(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	referenceImage := image.NewRGBA(image.Rect(0, 0, 3, 1))
	actualImage := image.NewRGBA(image.Rect(0, 0, 2, 1))
	referenceImage.SetRGBA(1, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	referenceImage.SetRGBA(2, 0, color.RGBA{R: 20, G: 30, B: 40, A: 255})
	actualImage.SetRGBA(0, 0, color.RGBA{R: 11, G: 21, B: 31, A: 255})
	actualImage.SetRGBA(1, 0, color.RGBA{R: 21, G: 31, B: 41, A: 255})
	writeTestPNG(t, reference, referenceImage)
	writeTestPNG(t, actual, actualImage)

	result := executeCommand(NewCommand(), "scan", reference, actual, "--reference-crop", "1,0,2,1", "--actual-crop", "0,0,2,1", "--y", "0")

	require.NoError(t, result.Err)
	assert.Equal(t, "image,axis,index,start,end,length,hex,input_axis,input_index\nref,x,0,0,0,1,#0A141E,x,0\nref,x,0,1,1,1,#141E28,x,0\nact,x,0,0,0,1,#0B151F,x,0\nact,x,0,1,1,1,#151F29,x,0\n", result.Stdout)
}

func TestScanCommandValidatesOptionsBeforeReadingImages(t *testing.T) {
	tests := []struct {
		name, expected string
		args           []string
	}{
		{name: "missing axis", expected: "provide exactly one of --x/--column or --y/--row"},
		{name: "multiple axes", args: []string{"--x", "0", "--y", "0"}, expected: "provide exactly one of --x/--column or --y/--row"},
		{name: "negative axis", args: []string{"--x", "-1"}, expected: "--x must be non-negative"},
		{name: "invalid format", args: []string{"--x", "0", "--format", "yaml"}, expected: `invalid --format "yaml": expected csv or json`},
		{name: "invalid limit", args: []string{"--x", "0", "--limit", "0"}, expected: "--limit must be greater than zero"},
		{name: "full and limit", args: []string{"--x", "0", "--full", "--limit", "5"}, expected: "--full cannot be combined with --limit"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"scan", "missing-reference.png", "missing-actual.png"}, test.args...)
			result := executeCommand(NewCommand(), args...)

			require.Error(t, result.Err)
			assert.ErrorContains(t, result.Err, test.expected)
			assert.Equal(t, 2, clipkg.ExitCode(result.Err))
			assert.NotContains(t, result.Err.Error(), "decode reference")
		})
	}
}

func TestScanCommandAcceptsRowAndColumnAliases(t *testing.T) {
	directory := t.TempDir()
	referencePath := filepath.Join(directory, "reference.png")
	actualPath := filepath.Join(directory, "actual.png")
	writeTestPNG(t, referencePath, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	writeTestPNG(t, actualPath, image.NewRGBA(image.Rect(0, 0, 2, 2)))

	row := executeCommand(NewCommand(), "scan", referencePath, actualPath, "--row", "0", "--format", "json")
	column := executeCommand(NewCommand(), "scan", referencePath, actualPath, "--column", "0", "--format", "json")

	require.NoError(t, row.Err)
	require.NoError(t, column.Err)
	assert.Contains(t, row.Stdout, `"axis": "x"`)
	assert.Contains(t, column.Stdout, `"axis": "y"`)
}
