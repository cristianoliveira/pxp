// Package annotationio owns JSON persistence for annotation documents.
package annotationio

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/cristianoliveira/pxp/internal/annotations"
	"github.com/cristianoliveira/pxp/internal/artifact"
)

type documentDTO struct {
	Version         int             `json:"version"`
	CoordinateSpace sizeDTO         `json:"coordinateSpace"`
	Annotations     []annotationDTO `json:"annotations"`
}
type sizeDTO struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}
type boundsDTO struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}
type annotationDTO struct {
	ID       string         `json:"id"`
	Label    string         `json:"label,omitempty"`
	Bounds   boundsDTO      `json:"bounds"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func toDTO(document annotations.Document) documentDTO {
	return documentDTO{Version: document.Version, CoordinateSpace: sizeDTO{Width: document.CoordinateSpace.Width, Height: document.CoordinateSpace.Height}, Annotations: func() []annotationDTO {
		var items []annotationDTO
		if document.Annotations != nil {
			items = make([]annotationDTO, len(document.Annotations))
		}
		for index, annotation := range document.Annotations {
			items[index] = annotationDTO{ID: annotation.ID, Label: annotation.Label, Bounds: boundsDTO{X: annotation.Bounds.X, Y: annotation.Bounds.Y, Width: annotation.Bounds.Width, Height: annotation.Bounds.Height}, Metadata: annotation.Metadata}
		}
		return items
	}()}
}

func fromDTO(document documentDTO) annotations.Document {
	result := annotations.Document{Version: document.Version, CoordinateSpace: annotations.Size{Width: document.CoordinateSpace.Width, Height: document.CoordinateSpace.Height}}
	if document.Annotations != nil {
		result.Annotations = make([]annotations.Annotation, len(document.Annotations))
	}
	for index, annotation := range document.Annotations {
		result.Annotations[index] = annotations.Annotation{ID: annotation.ID, Label: annotation.Label, Bounds: annotations.Bounds{X: annotation.Bounds.X, Y: annotation.Bounds.Y, Width: annotation.Bounds.Width, Height: annotation.Bounds.Height}, Metadata: annotation.Metadata}
	}
	return result
}

// Write persists an annotation document as indented JSON.
func Write(path string, document annotations.Document) error {
	file, err := artifact.CreateFile(path)
	if err != nil {
		return fmt.Errorf("write annotations: %w", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(toDTO(document))
	closeErr := file.Close()
	if encodeErr != nil {
		return fmt.Errorf("write annotations: %w", encodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("write annotations: %w", closeErr)
	}
	return nil
}

// Load decodes and validates an annotation document from JSON.
func Load(path string) (*annotations.Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read annotations: %w", err)
	}
	defer func() { _ = file.Close() }()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var wire documentDTO
	if err := decoder.Decode(&wire); err != nil {
		return nil, fmt.Errorf("decode annotations: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return nil, err
	}
	document := fromDTO(wire)
	if err := document.Validate(); err != nil {
		return nil, err
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
