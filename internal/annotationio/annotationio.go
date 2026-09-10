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

// Write persists an annotation document as indented JSON.
func Write(path string, document annotations.Document) error {
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

// Load decodes and validates an annotation document from JSON.
func Load(path string) (*annotations.Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read annotations: %w", err)
	}
	defer func() { _ = file.Close() }()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var document annotations.Document
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode annotations: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return nil, err
	}
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
