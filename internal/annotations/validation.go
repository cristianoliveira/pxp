package annotations

import "fmt"

// Validate checks document rules independently of serialization and storage.
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
