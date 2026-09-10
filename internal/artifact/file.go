// Package artifact owns filesystem persistence for generated artifacts.
package artifact

import (
	"io/fs"
	"os"
	"path/filepath"
)

// CreateFile creates path and any missing parent directories.
func CreateFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.Create(path)
}

// WriteFile writes data to path after creating any missing parent directories.
func WriteFile(path string, data []byte, permission fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, permission)
}
