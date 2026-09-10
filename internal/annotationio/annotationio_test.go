package annotationio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cristianoliveira/pxp/internal/annotations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRejectsMissingMalformedAndTrailingJSON(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	assert.ErrorContains(t, err, "read annotations")

	path := filepath.Join(t.TempDir(), "invalid.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"version":1`), 0o600))
	_, err = Load(path)
	assert.ErrorContains(t, err, "decode annotations")

	require.NoError(t, os.WriteFile(path, []byte(`{"version":1,"coordinateSpace":{"width":1,"height":1},"annotations":[]} {}`), 0o600))
	_, err = Load(path)
	assert.EqualError(t, err, "decode annotations: multiple JSON values")
}

func TestWriteCreatesParentAndRoundTripsDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "annotations.json")
	document := annotations.Document{Version: 1, CoordinateSpace: annotations.Size{Width: 10, Height: 10}}

	require.NoError(t, Write(path, document))
	loaded, err := Load(path)

	require.NoError(t, err)
	assert.Equal(t, document, *loaded)
}
