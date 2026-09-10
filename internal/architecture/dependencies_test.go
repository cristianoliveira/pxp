package architecture

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDomainPackagesHaveNoPersistenceDependencies prevents file and codec
// mechanisms from leaking into deterministic domain packages.
func TestDomainPackagesHaveNoPersistenceDependencies(t *testing.T) {
	_, sourceFile, _, _ := runtime.Caller(0)
	internalRoot := filepath.Join(filepath.Dir(sourceFile), "..")
	for _, packageName := range []string{"annotations", "imagediff"} {
		files, err := filepath.Glob(filepath.Join(internalRoot, packageName, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			for _, imported := range file.Imports {
				path := strings.Trim(imported.Path.Value, `"`)
				switch path {
				case "encoding/json", "io", "os", "path/filepath", "image/png", "github.com/cristianoliveira/pxp/internal/artifact", "github.com/cristianoliveira/pxp/internal/annotationio", "github.com/cristianoliveira/pxp/internal/imageio":
					t.Errorf("%s domain imports persistence mechanism %q", packageName, path)
				}
			}
		}
	}
}
