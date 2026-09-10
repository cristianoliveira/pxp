package architecture

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

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
			imports := packageImports(t, path)
			for _, forbidden := range []string{"encoding/json", "io", "os", "path/filepath", "image/png", "github.com/cristianoliveira/pxp/internal/artifact", "github.com/cristianoliveira/pxp/internal/annotationio", "github.com/cristianoliveira/pxp/internal/imageio"} {
				if imports[forbidden] {
					t.Errorf("%s domain imports persistence mechanism %q", packageName, forbidden)
				}
			}
		}
	}
}

func TestCompositionUsesAdaptersAtCommandBoundary(t *testing.T) {
	_, sourceFile, _, _ := runtime.Caller(0)
	internalRoot := filepath.Join(filepath.Dir(sourceFile), "..")
	imports := packageImports(t, filepath.Join(internalRoot, "commands", "compare.go"))
	assertImport(t, imports, "github.com/cristianoliveira/pxp/internal/imageio")
	assertImport(t, imports, "github.com/cristianoliveira/pxp/internal/annotationio")
}

func packageImports(t *testing.T, path string) map[string]bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	imports := make(map[string]bool, len(file.Imports))
	for _, imported := range file.Imports {
		imports[strings.Trim(imported.Path.Value, `"`)] = true
	}
	return imports
}

func assertImport(t *testing.T, imports map[string]bool, path string) {
	t.Helper()
	if !imports[path] {
		t.Errorf("composition boundary does not import %q", path)
	}
}
