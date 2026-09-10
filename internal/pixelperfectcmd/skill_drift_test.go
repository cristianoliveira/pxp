package pixelperfectcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPixelPerfectHomeAndSkillGuidanceStaySynchronized(t *testing.T) {
	skill, err := os.ReadFile(filepath.Join("..", "..", "skills", "pxp", "SKILL.md"))
	require.NoError(t, err)
	guidance := string(skill)
	result := executeCommand(newCommandWithExecutable(func() (string, error) { return "~/bin/pxp", nil }))
	require.NoError(t, result.Err)

	for _, example := range []string{
		"pxp reference.png actual.png",
		"pxp probe",
		"pxp scan",
	} {
		assert.Containsf(t, result.Stdout, example, "home view missing %q", example)
		assert.Containsf(t, guidance, example, "skill drift: home example %q is missing", example)
	}
	assert.Contains(t, guidance, "25 points")
	assert.Contains(t, guidance, "25 runs per image")
}
