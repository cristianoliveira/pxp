package annotations

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchJSONPreservesStructuredOutputKeys(t *testing.T) {
	encoded, err := json.Marshal(Match{ID: "sidebar", Label: "Sidebar", RegionIntersectionRatio: 0.5, AnnotationIntersectionRatio: 1, Metadata: map[string]any{"source": "capture"}})

	require.NoError(t, err)
	require.JSONEq(t, `{"id":"sidebar","label":"Sidebar","regionIntersectionRatio":0.5,"annotationIntersectionRatio":1,"metadata":{"source":"capture"}}`, string(encoded))
}
