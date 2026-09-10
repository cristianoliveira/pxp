package annotations

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/cristianoliveira/pxp/internal/artifact"
)

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Bounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Annotation struct {
	ID       string         `json:"id"`
	Label    string         `json:"label,omitempty"`
	Bounds   Bounds         `json:"bounds"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type Document struct {
	Version         int          `json:"version"`
	CoordinateSpace Size         `json:"coordinateSpace"`
	Annotations     []Annotation `json:"annotations"`
}

type Match struct {
	ID                          string         `json:"id"`
	Label                       string         `json:"label,omitempty"`
	RegionIntersectionRatio     float64        `json:"regionIntersectionRatio"`
	AnnotationIntersectionRatio float64        `json:"annotationIntersectionRatio"`
	Metadata                    map[string]any `json:"metadata,omitempty"`
}

func Write(path string, document Document) error {
	file, err := artifact.CreateFile(path)
	if err != nil {
		return fmt.Errorf("write annotations: %w", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(document)
	closeErr := file.Close()
	if encodeErr != nil {
		return fmt.Errorf("write annotations: %w", encodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("write annotations: %w", closeErr)
	}
	return nil
}

func Load(path string) (*Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read annotations: %w", err)
	}
	defer func() { _ = file.Close() }()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var document Document
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode annotations: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return nil, err
	}
	if document.Version != 1 {
		return nil, fmt.Errorf("unsupported annotations version %d", document.Version)
	}
	if document.CoordinateSpace.Width < 1 || document.CoordinateSpace.Height < 1 {
		return nil, fmt.Errorf("annotation coordinate space dimensions must be positive")
	}
	seen := map[string]bool{}
	for _, annotation := range document.Annotations {
		if annotation.ID == "" {
			return nil, fmt.Errorf("annotation id must not be empty")
		}
		if seen[annotation.ID] {
			return nil, fmt.Errorf("duplicate annotation id %q", annotation.ID)
		}
		seen[annotation.ID] = true
		if !within(annotation.Bounds, document.CoordinateSpace) {
			return nil, fmt.Errorf("annotation %q bounds are outside coordinate space", annotation.ID)
		}
	}
	return &document, nil
}

func rejectTrailingJSON(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode annotations: multiple JSON values")
		}
		return fmt.Errorf("decode annotations: %w", err)
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
		matches = append(matches, Match{
			ID: annotation.ID, Label: annotation.Label, Metadata: annotation.Metadata,
			RegionIntersectionRatio:     float64(area) / float64(region.Width*region.Height),
			AnnotationIntersectionRatio: float64(area) / float64(annotation.Bounds.Width*annotation.Bounds.Height),
		})
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
