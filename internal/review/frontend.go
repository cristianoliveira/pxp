package review

import (
	"embed"
	"io/fs"
)

// frontendAssets contains the review UI shipped inside the pxp binary.
// Keeping the browser code as files makes it independently readable and testable
// while preserving the single-binary, offline review workflow.
//
//go:embed frontend/index.html frontend/style.css frontend/app.js
var frontendAssets embed.FS

func frontendFile(name string) ([]byte, error) {
	return fs.ReadFile(frontendAssets, "frontend/"+name)
}
