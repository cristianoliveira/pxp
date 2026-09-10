package imagediff

import "image"

// DecodedImages contains invocation-scoped normalized image data.
type DecodedImages struct {
	Reference *image.NRGBA
	Actual    *image.NRGBA
}
