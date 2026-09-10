package commands

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/cristianoliveira/pxp/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComparisonProfileAppliesValuesAndReportsSources(t *testing.T) {
	dir := t.TempDir()
	reference, actual := filepath.Join(dir, "reference.png"), filepath.Join(dir, "actual.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 5, 3)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 5, 3)))
	profile := filepath.Join(dir, "profile.json")
	require.NoError(t, os.WriteFile(profile, []byte(`{"version":1,"suggestOffset":2,"regionGap":3,"minRegionPixels":2}`), 0o600))

	result := executeCommand(NewCommand(), reference, actual, "--profile", profile)

	require.NoError(t, result.Err)
	var output outputEnvelope
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
	require.NotNil(t, output.Configuration)
	assert.Equal(t, resolvedValue[int]{Value: 2, Source: "profile"}, output.Configuration.Resolved.SuggestOffset)
	assert.Equal(t, resolvedValue[int]{Value: 3, Source: "profile"}, output.Configuration.Resolved.RegionGap)
	assert.Equal(t, resolvedValue[int]{Value: 2, Source: "profile"}, output.Configuration.Resolved.MinRegionPixels)
}

func TestExplicitFlagOverridesComparisonProfile(t *testing.T) {
	dir := t.TempDir()
	reference, actual := filepath.Join(dir, "reference.png"), filepath.Join(dir, "actual.png")
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	writeTestPNG(t, actual, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	profile := filepath.Join(dir, "profile.json")
	require.NoError(t, os.WriteFile(profile, []byte(`{"version":1,"suggestOffset":8}`), 0o600))

	result := executeCommand(NewCommand(), reference, actual, "--profile", profile, "--suggest-offset", "0")

	require.NoError(t, result.Err)
	var output outputEnvelope
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
	assert.Equal(t, resolvedValue[int]{Value: 0, Source: "flag"}, output.Configuration.Resolved.SuggestOffset)
}

func TestComparisonProfileRejectsUnknownFieldsAndVersions(t *testing.T) {
	dir := t.TempDir()
	unknown := filepath.Join(dir, "unknown.json")
	require.NoError(t, os.WriteFile(unknown, []byte(`{"version":1,"magic":2}`), 0o600))
	result := executeCommand(NewCommand(), "reference.png", "actual.png", "--profile", unknown)
	assert.ErrorContains(t, result.Err, `unknown field "magic"`)
	assert.Equal(t, 2, cli.ExitCode(result.Err))

	unsupported := filepath.Join(dir, "unsupported.json")
	require.NoError(t, os.WriteFile(unsupported, []byte(`{"version":2}`), 0o600))
	result = executeCommand(NewCommand(), "reference.png", "actual.png", "--profile", unsupported)
	assert.EqualError(t, result.Err, "unsupported --profile version 2")
	assert.Equal(t, 2, cli.ExitCode(result.Err))

	missing := executeCommand(NewCommand(), "reference.png", "actual.png", "--profile", filepath.Join(dir, "missing.json"))
	assert.Equal(t, 1, cli.ExitCode(missing.Err))
}
