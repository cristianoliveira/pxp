package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveExecutablePathHandlesInvocationForms(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(bin, 0o755))
	target := filepath.Join(bin, "pxp")
	require.NoError(t, os.WriteFile(target, []byte("binary"), 0o755))
	link := filepath.Join(t.TempDir(), "pxp-link")
	require.NoError(t, os.Symlink(target, link))
	t.Setenv("PATH", bin)
	pathResolved, err := exec.LookPath("pxp")
	require.NoError(t, err)

	tests := []struct {
		name, executable, expected string
	}{
		{name: "PATH resolved absolute", executable: pathResolved, expected: "~/bin/pxp"},
		{name: "absolute invocation", executable: target, expected: "~/bin/pxp"},
		{name: "symlink", executable: link, expected: "~/bin/pxp"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path, err := ResolveExecutablePath(test.executable, home)
			require.NoError(t, err)
			assert.Equal(t, test.expected, path)
		})
	}
}

func TestCollapseHomePathRequiresDirectoryBoundary(t *testing.T) {
	assert.Equal(t, "~", CollapseHomePath("/Users/agent", "/Users/agent"))
	assert.Equal(t, "~/bin/tool", CollapseHomePath("/Users/agent/bin/tool", "/Users/agent"))
	assert.Equal(t, "/Users/agent-other/tool", CollapseHomePath("/Users/agent-other/tool", "/Users/agent"))
}
