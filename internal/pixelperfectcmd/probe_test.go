package pixelperfectcmd

import (
	"encoding/json"
	"image"
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	clipkg "github.com/cristianoliveira/pxp/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeCommandReportsPointColorsAndDelta(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	referenceImage := image.NewRGBA(image.Rect(0, 0, 2, 2))
	actualImage := image.NewRGBA(image.Rect(0, 0, 2, 2))
	referenceImage.SetRGBA(1, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	actualImage.SetRGBA(1, 0, color.RGBA{R: 244, G: 244, B: 244, A: 255})
	writeTestPNG(t, reference, referenceImage)
	writeTestPNG(t, actual, actualImage)

	result := executeCommand(NewCommand(), "probe", reference, actual, "--at", "1,0")

	require.NoError(t, result.Err)
	assert.Equal(t, "x,y,ref,act,delta,input_ref,input_act\n1,0,#FFFFFF,#F4F4F4,11,,\n", result.Stdout)
}

func TestProbeCommandSamplesInclusiveLineWithStep(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 5, 5)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 5, 5)))

	result := executeCommand(NewCommand(), "probe", reference, actual, "--from", "0,0", "--to", "4,4", "--step", "2")

	require.NoError(t, result.Err)
	assert.Equal(t, "x,y,ref,act,delta,input_ref,input_act\n0,0,#000000,#000000,0,,\n2,2,#000000,#000000,0,,\n4,4,#000000,#000000,0,,\n", result.Stdout)
}

func TestProbeCommandExpandsAndDeduplicatesRadius(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 5, 3)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 5, 3)))

	result := executeCommand(NewCommand(), "probe", reference, actual, "--at", "2,1", "--from", "1,1", "--to", "3,1", "--radius", "1", "--format", "json")

	require.NoError(t, result.Err)
	var output probeOutput
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
	assert.Len(t, output.Points, 15)
}

func TestProbeCommandBoundsOutputAndProvidesScopePreservingHint(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference image.png")
	actual := filepath.Join(dir, "actual.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 11, 11)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 11, 11)))

	result := executeCommand(NewCommand(), "probe", reference, actual, "--at", "5,5", "--radius", "5", "--format", "json")

	require.NoError(t, result.Err)
	var output probeOutput
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
	assert.Equal(t, 121, output.Total)
	assert.Equal(t, 25, output.Returned)
	assert.True(t, output.Truncated)
	assert.Len(t, output.Points, 25)
	assert.Contains(t, output.Hint, "pxp probe")
	assert.Contains(t, output.Hint, "'"+reference+"'")
	assert.Contains(t, output.Hint, "--at 5,5")
	assert.Contains(t, output.Hint, "--radius 5")
	assert.Contains(t, output.Hint, "--format json")
	assert.Contains(t, output.Hint, "--full")

	csv := executeCommand(NewCommand(), "probe", reference, actual, "--at", "5,5", "--radius", "5")
	require.NoError(t, csv.Err)
	assert.Len(t, strings.Split(strings.TrimSpace(csv.Stdout), "\n"), 27)
	assert.Contains(t, csv.Stdout, "# total=121 returned=25 truncated=true")
	assert.Contains(t, csv.Stdout, "--full")

	full := executeCommand(NewCommand(), "probe", reference, actual, "--at", "5,5", "--radius", "5", "--format", "json", "--full")
	require.NoError(t, full.Err)
	output = probeOutput{}
	require.NoError(t, json.Unmarshal([]byte(full.Stdout), &output))
	assert.Equal(t, 121, output.Total)
	assert.Equal(t, 121, output.Returned)
	assert.False(t, output.Truncated)
	assert.Empty(t, output.Hint)
}

func TestProbeCommandRejectsIncompleteLineAndInvalidOptions(t *testing.T) {
	tests := []struct {
		name, expected string
		args           []string
	}{
		{name: "incomplete line", args: []string{"--from", "0,0"}, expected: "--from and --to must be provided together"},
		{name: "invalid step", args: []string{"--from", "0,0", "--to", "1,1", "--step", "0"}, expected: "--step must be positive"},
		{name: "invalid radius", args: []string{"--at", "0,0", "--radius", "-1"}, expected: "--radius must be non-negative"},
		{name: "invalid format", args: []string{"--at", "0,0", "--format", "yaml"}, expected: `invalid --format "yaml": expected csv or json`},
		{name: "invalid limit", args: []string{"--at", "0,0", "--limit", "0"}, expected: "--limit must be greater than zero"},
		{name: "full and limit", args: []string{"--at", "0,0", "--full", "--limit", "5"}, expected: "--full cannot be combined with --limit"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"probe", "missing-reference.png", "missing-actual.png"}, test.args...)
			result := executeCommand(NewCommand(), args...)

			require.Error(t, result.Err)
			assert.ErrorContains(t, result.Err, test.expected)
			assert.Equal(t, 2, clipkg.ExitCode(result.Err))
			assert.NotContains(t, result.Err.Error(), "decode reference")
		})
	}
}

func TestProbeCommandWritesJSONFormat(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	referenceImage := image.NewRGBA(image.Rect(0, 0, 2, 1))
	actualImage := image.NewRGBA(image.Rect(0, 0, 2, 1))
	referenceImage.SetRGBA(1, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	actualImage.SetRGBA(1, 0, color.RGBA{R: 244, G: 244, B: 244, A: 255})
	writeTestPNG(t, reference, referenceImage)
	writeTestPNG(t, actual, actualImage)

	result := executeCommand(NewCommand(), "probe", reference, actual, "--at", "1,0", "--format", "json")

	require.NoError(t, result.Err)
	assert.JSONEq(t, `{"total":1,"returned":1,"truncated":false,"points":[{"point":{"x":1,"y":0},"reference":{"rgba":[255,255,255,255],"hex":"#FFFFFF"},"actual":{"rgba":[244,244,244,255],"hex":"#F4F4F4"},"delta":{"r":11,"g":11,"b":11,"a":0}}]}`, result.Stdout)
}

func TestProbeCommandAppliesInputCrops(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	referenceImage := image.NewRGBA(image.Rect(0, 0, 3, 1))
	actualImage := image.NewRGBA(image.Rect(0, 0, 2, 1))
	referenceImage.SetRGBA(1, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	actualImage.SetRGBA(0, 0, color.RGBA{R: 11, G: 21, B: 31, A: 255})
	writeTestPNG(t, reference, referenceImage)
	writeTestPNG(t, actual, actualImage)

	result := executeCommand(NewCommand(), "probe", reference, actual, "--reference-crop", "1,0,2,1", "--actual-crop", "0,0,2,1", "--at", "0,0", "--at", "1,0")

	require.NoError(t, result.Err)
	assert.Equal(t, "x,y,ref,act,delta,input_ref,input_act\n0,0,#0A141E,#0B151F,1,1:0,0:0\n1,0,#000000,#000000,0,2:0,1:0\n", result.Stdout)
}

func TestProbeCommandRejectsOutOfBoundsPoint(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 2, 2)))

	result := executeCommand(NewCommand(), "probe", reference, actual, "--at", "2,0")

	require.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "--at point 2,0 is outside image bounds 2x2")
}

func TestProbeCommandRejectsDimensionMismatch(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 3, 2)))

	result := executeCommand(NewCommand(), "probe", reference, actual, "--at", "1,0")

	require.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "image dimensions differ")
}
