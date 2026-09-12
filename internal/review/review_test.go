package review

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSessionRejectsInvalidPerceptualThreshold(t *testing.T) {
	_, err := NewSession("reference.png", "actual.png", t.TempDir(), "", 0, -1)
	require.EqualError(t, err, "perceptual threshold must be a finite non-negative number")
}

func TestSessionPersistsImmutableSnapshotAndFeedback(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.RGBA{R: 20, A: 255}, color.RGBA{R: 40, A: 255})
	session, err := NewSession(reference, actual, filepath.Join(dir, "rounds"), "", 0, 0.1)
	require.NoError(t, err)
	before, err := os.ReadFile(session.snapshot.Actual.Path)
	require.NoError(t, err)
	require.Equal(t, 3, session.snapshot.Actual.Width)

	// The source can change after the server starts; the review remains tied to its copy.
	require.NoError(t, os.WriteFile(actual, []byte("not the reviewed image"), 0o600))
	after, err := os.ReadFile(session.snapshot.Actual.Path)
	require.NoError(t, err)
	require.Equal(t, before, after)

	result, err := session.Submit(FeedbackRequest{
		Decision:    "submitted",
		Notes:       "move the button down",
		Annotations: []Annotation{{Image: "actual", Type: "rectangle", X: 1, Y: 1, Width: 2, Height: 1, Note: "button"}},
	})
	require.NoError(t, err)
	require.Equal(t, "note-0001", mustReadFeedback(t, result.FeedbackPath).Annotations[0].ID)
	require.Equal(t, "submitted", mustReadFeedback(t, result.FeedbackPath).Decision)
	_, err = session.Submit(FeedbackRequest{Decision: "approved"})
	require.Error(t, err)
}

func TestHandlerUsesOriginalPixelCoordinatesAndExplicitDecision(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.Black, color.White)
	session, err := NewSession(reference, actual, dir, "", 0, 0.1)
	require.NoError(t, err)
	server := httptest.NewServer(sessionHandler(session))
	defer server.Close()

	response, err := http.Get(server.URL + "/api/session")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "nosniff", response.Header.Get("X-Content-Type-Options"))
	require.NoError(t, response.Body.Close())

	body := `{"decision":"approved","notes":"looks good","annotations":[{"image":"actual","type":"point","x":2,"y":1,"note":"edge"},{"image":"reference","type":"rectangle","x":1,"y":0,"width":2,"height":2}]}`
	response, err = http.Post(server.URL+"/api/feedback", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var result Result
	require.NoError(t, json.NewDecoder(response.Body).Decode(&result))
	require.NoError(t, response.Body.Close())
	feedback := mustReadFeedback(t, result.FeedbackPath)
	require.Equal(t, "approved", feedback.Decision)
	require.Equal(t, "note-0001", feedback.Annotations[0].ID)
	require.Equal(t, 2, feedback.Annotations[0].X)
	require.Equal(t, 1, feedback.Annotations[0].Y)
	require.Equal(t, 2, feedback.Annotations[1].Width)

	response, err = http.Post(server.URL+"/api/feedback", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.NoError(t, response.Body.Close())
}

func TestHandlerRejectsOutOfBoundsAndUnknownFields(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.Black, color.White)
	session, err := NewSession(reference, actual, dir, "", 0, 0.1)
	require.NoError(t, err)
	server := httptest.NewServer(sessionHandler(session))
	defer server.Close()
	for _, body := range []string{
		`{"decision":"submitted","annotations":[{"image":"actual","type":"point","x":3,"y":0}]}`,
		`{"decision":"submitted","unexpected":true}`,
		`{"decision":"nope"}`,
	} {
		response, postErr := http.Post(server.URL+"/api/feedback", "application/json", bytes.NewBufferString(body))
		require.NoError(t, postErr)
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
		require.NoError(t, response.Body.Close())
	}
}

func TestNewSessionLinksPreviousRoundWithoutOverwritingIt(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.Black, color.White)
	first, err := NewSession(reference, actual, dir, "", 0, 0.1)
	require.NoError(t, err)
	firstResult, err := first.Submit(FeedbackRequest{Decision: "submitted", Notes: "fix it"})
	require.NoError(t, err)
	firstBytes, err := os.ReadFile(firstResult.FeedbackPath)
	require.NoError(t, err)

	second, err := NewSession(reference, actual, dir, firstResult.FeedbackPath, 0, 0.1)
	require.NoError(t, err)
	secondResult, err := second.Submit(FeedbackRequest{Decision: "approved"})
	require.NoError(t, err)
	require.Equal(t, 2, secondResult.Round)
	require.NotEqual(t, firstResult.FeedbackPath, secondResult.FeedbackPath)
	require.Equal(t, firstBytes, mustRead(t, firstResult.FeedbackPath))
	feedback := mustReadFeedback(t, secondResult.FeedbackPath)
	require.Equal(t, filepath.Join(second.Root(), "previous-feedback.json"), feedback.PreviousFeedback)
}

func TestServerLifecycleIsLoopbackOnly(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.Black, color.White)
	session, err := NewSession(reference, actual, dir, "", 0, 0.1)
	require.NoError(t, err)
	server := NewServer(session)
	url, err := server.Start()
	require.NoError(t, err)
	defer server.Close()
	require.Contains(t, url, "http://127.0.0.1:")
	response, err := http.Post(url+"/api/feedback", "application/json", bytes.NewBufferString(`{"decision":"approved"}`))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, response.Body.Close())
	require.Equal(t, "approved", server.Wait().Decision)
}

func sessionHandler(session *Session) http.Handler { return NewServer(session).Handler() }
func mustReadFeedback(t *testing.T, path string) Feedback {
	t.Helper()
	var value Feedback
	require.NoError(t, json.Unmarshal(mustRead(t, path), &value))
	return value
}
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	value, err := os.ReadFile(path)
	require.NoError(t, err)
	return value
}
func writePNG(t *testing.T, dir, name string, reference, actual color.Color) (string, string) {
	t.Helper()
	ref := image.NewRGBA(image.Rect(0, 0, 3, 2))
	act := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			ref.Set(x, y, reference)
			act.Set(x, y, actual)
		}
	}
	refPath := filepath.Join(dir, name)
	actPath := filepath.Join(dir, "actual.png")
	writeImage(t, refPath, ref)
	writeImage(t, actPath, act)
	return refPath, actPath
}
func writeImage(t *testing.T, path string, value image.Image) {
	t.Helper()
	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(file, value))
	require.NoError(t, file.Close())
}
