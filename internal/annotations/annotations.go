package annotations

import "fmt"

type Size struct {
	Width  int
	Height int
}
type Bounds struct {
	X      int
	Y      int
	Width  int
	Height int
}
type Annotation struct {
	ID       string
	Label    string
	Bounds   Bounds
	Metadata map[string]any
}
type Document struct {
	Version         int
	CoordinateSpace Size
	Annotations     []Annotation
}
type Match struct {
	ID                          string
	Label                       string
	RegionIntersectionRatio     float64
	AnnotationIntersectionRatio float64
	Metadata                    map[string]any
}

// Validate checks the domain invariants without reading or writing external data.
func (document Document) Validate() error {
	if document.Version != 1 {
		return fmt.Errorf("unsupported annotations version %d", document.Version)
	}
	if document.CoordinateSpace.Width < 1 || document.CoordinateSpace.Height < 1 {
		return fmt.Errorf("annotation coordinate space dimensions must be positive")
	}
	seen := map[string]bool{}
	for _, annotation := range document.Annotations {
		if annotation.ID == "" {
			return fmt.Errorf("annotation id must not be empty")
		}
		if seen[annotation.ID] {
			return fmt.Errorf("duplicate annotation id %q", annotation.ID)
		}
		seen[annotation.ID] = true
		if !within(annotation.Bounds, document.CoordinateSpace) {
			return fmt.Errorf("annotation %q bounds are outside coordinate space", annotation.ID)
		}
	}
	return nil
}

func (document Document) ValidateDimensions(width, height int) error {
	if document.CoordinateSpace.Width != width || document.CoordinateSpace.Height != height {
		return fmt.Errorf("annotation coordinate space %dx%d does not match comparison image %dx%d", document.CoordinateSpace.Width, document.CoordinateSpace.Height, width, height)
	}
	return nil
}

func (document Document) Intersections(region Bounds) []Match {
	matches := make([]Match, 0)
	for _, annotation := range document.Annotations {
		intersection := intersect(region, annotation.Bounds)
		area := intersection.Width * intersection.Height
		if area == 0 {
			continue
		}
		matches = append(matches, Match{ID: annotation.ID, Label: annotation.Label, Metadata: annotation.Metadata, RegionIntersectionRatio: float64(area) / float64(region.Width*region.Height), AnnotationIntersectionRatio: float64(area) / float64(annotation.Bounds.Width*annotation.Bounds.Height)})
	}
	return matches
}

func within(bounds Bounds, size Size) bool {
	return bounds.X >= 0 && bounds.Y >= 0 && bounds.Width > 0 && bounds.Height > 0 && bounds.X+bounds.Width <= size.Width && bounds.Y+bounds.Height <= size.Height
}

func intersect(first, second Bounds) Bounds {
	x, y := max(first.X, second.X), max(first.Y, second.Y)
	endX, endY := min(first.X+first.Width, second.X+second.Width), min(first.Y+first.Height, second.Y+second.Height)
	return Bounds{X: x, Y: y, Width: max(0, endX-x), Height: max(0, endY-y)}
}
