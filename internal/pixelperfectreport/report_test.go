package pixelperfectreport

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderEscapesUserProvidedPathsAndEmbedsImages(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, `reference-<script>.png`)
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	writePNG(t, reference)
	writePNG(t, actual)
	writePNG(t, mask)

	html, err := Render(Input{
		ReferencePath: reference,
		ActualPath:    actual,
		MaskPath:      mask,
		Result:        imagediff.ImageComparison{ChangedPixels: 1, RMSE: 0.5},
	})

	require.NoError(t, err)
	content := string(html)
	assert.Contains(t, content, "Pixel Perfect Report")
	assert.Contains(t, content, "data:image/png;base64,")
	assert.Contains(t, content, "reference-&lt;script&gt;.png")
	assert.NotContains(t, content, `reference-<script>.png`)
}

func TestWriteCreatesMissingParentDirectories(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	writePNG(t, reference)
	writePNG(t, actual)
	writePNG(t, mask)
	path := filepath.Join(dir, "missing", "reports", "report.html")

	err := Write(path, Input{ReferencePath: reference, ActualPath: actual, MaskPath: mask})

	require.NoError(t, err)
	assert.FileExists(t, path)
}

func TestRenderIncludesAdvisoryRegionMovements(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "mask.png")
	writePNG(t, reference)
	writePNG(t, actual)
	writePNG(t, mask)

	html, err := Render(Input{
		ReferencePath: reference,
		ActualPath:    actual,
		MaskPath:      mask,
		Result: imagediff.ImageComparison{MovedRegions: []imagediff.RegionMovement{{
			Bounds:     imagediff.Bounds{X: 1, Y: 2, Width: 3, Height: 4},
			DX:         5,
			DY:         -1,
			Confidence: 0.75,
		}}},
	})

	require.NoError(t, err)
	content := string(html)
	assert.Contains(t, content, "Advisory region movements")
	assert.Contains(t, content, "dx=5 dy=-1")
	assert.Contains(t, content, "confidence=0.75")
}

func writePNG(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(file, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	require.NoError(t, file.Close())
}
