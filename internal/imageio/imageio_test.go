package imageio

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/stretchr/testify/require"
)

func TestCompareImagesWritesMaskAtAdapterBoundary(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference.png")
	actual := filepath.Join(dir, "actual.png")
	mask := filepath.Join(dir, "nested", "mask.png")
	writePNG(t, reference, image.NewNRGBA(image.Rect(0, 0, 1, 1)))
	changed := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	changed.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	writePNG(t, actual, changed)

	result, err := CompareImages(reference, actual, mask, 0)

	require.NoError(t, err)
	require.Equal(t, 1, result.ChangedPixels)
	require.Equal(t, mask, result.Mask)
	_, err = os.Stat(mask)
	require.NoError(t, err)
}

func TestIgnoredRegionsFromMaskKeepsDomainBoundsAtAdapterBoundary(t *testing.T) {
	dir := t.TempDir()
	mask := filepath.Join(dir, "mask.png")
	reference := filepath.Join(dir, "reference.png")
	maskImage := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	maskImage.SetNRGBA(0, 0, color.NRGBA{})
	maskImage.SetNRGBA(1, 0, color.NRGBA{R: 255, A: 255})
	writePNG(t, mask, maskImage)
	writePNG(t, reference, image.NewNRGBA(image.Rect(0, 0, 2, 1)))

	regions, err := IgnoredRegionsFromMask(mask, reference)

	require.NoError(t, err)
	require.Equal(t, []imagediff.Bounds{{X: 0, Y: 0, Width: 1, Height: 1}}, regions)
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(file, img))
	require.NoError(t, file.Close())
}
