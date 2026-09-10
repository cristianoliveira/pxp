package artifact

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFileCreatesMissingParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "nested", "result.txt")

	err := WriteFile(path, []byte("result"), 0o600)

	require.NoError(t, err)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "result", string(content))
}

func TestCreateFileCreatesMissingParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "nested", "result.txt")

	file, err := CreateFile(path)
	require.NoError(t, err)
	_, err = io.WriteString(file, "result")
	require.NoError(t, err)
	require.NoError(t, file.Close())

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "result", string(content))
}

func TestWriteFileReturnsParentCreationError(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "parent-file")
	require.NoError(t, os.WriteFile(parent, []byte("not a directory"), 0o600))

	err := WriteFile(filepath.Join(parent, "nested", "result.txt"), []byte("result"), 0o600)

	require.Error(t, err)
}

func TestWriteFileOverwritesContentAndPreservesExistingPermission(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.txt")

	require.NoError(t, WriteFile(path, []byte("before"), 0o600))
	require.NoError(t, WriteFile(path, []byte("after"), 0o644))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, "after", string(content))
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestCreateFileOverwritesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.txt")
	require.NoError(t, os.WriteFile(path, []byte("before"), 0o600))

	file, err := CreateFile(path)
	require.NoError(t, err)
	_, err = io.WriteString(file, "after")
	require.NoError(t, err)
	require.NoError(t, file.Close())

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "after", string(content))
}

func TestCreateFileReturnsParentCreationError(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "parent-file")
	require.NoError(t, os.WriteFile(parent, []byte("not a directory"), 0o600))

	file, err := CreateFile(filepath.Join(parent, "nested", "result.txt"))

	require.Nil(t, file)
	require.Error(t, err)
}
