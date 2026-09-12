package review

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSessionRejectsInvalidPerceptualThreshold(t *testing.T) {
	_, err := NewSession("reference.png", "actual.png", t.TempDir(), "", 0, -1)
	require.EqualError(t, err, "perceptual threshold must be a finite non-negative number")
}

func TestLoadContextNormalizesMultilineAndRejectsUnsafeShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "context.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
  "title": "  Spacing update  ",
  "what_changed": "First line\nSecond line",
  "what_to_test": "<script>alert(1)</script>",
  "expected_outcome": "  Buttons align  ",
  "limitations": "No mobile changes",
  "source_reference": "commit:abc123"
}`), 0o600))

	context, err := LoadContext(path)
	require.NoError(t, err)
	require.Equal(t, "Spacing update", context.Title)
	require.Equal(t, "First line\nSecond line", context.WhatChanged)
	require.Equal(t, "<script>alert(1)</script>", context.WhatToTest)
	require.Equal(t, "Buttons align", context.ExpectedOutcome)

	for _, content := range []string{
		`{"title":"ok","unknown":"field"}`,
		`{"title":"ok"}{"title":"trailing"}`,
		`{"title":"ok"}null`,
		`null`,
		`{"title":null}`,
		`{"title":123}`,
	} {
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		_, err = LoadContext(path)
		require.Error(t, err)
	}

	require.NoError(t, os.WriteFile(path, []byte(`{"title":""}`), 0o600))
	context, err = LoadContext(path)
	require.NoError(t, err)
	require.Equal(t, contextFallback, context.Title)
}

func TestLoadContextUsesDeterministicFallbackWhenOmitted(t *testing.T) {
	context, err := LoadContext("")
	require.NoError(t, err)
	require.Equal(t, contextFallback, context.Title)
	require.Empty(t, context.WhatChanged)

	path := filepath.Join(t.TempDir(), "empty-context.json")
	require.NoError(t, os.WriteFile(path, []byte(" \n\t"), 0o600))
	context, err = LoadContext(path)
	require.NoError(t, err)
	require.Equal(t, contextFallback, context.Title)
}

func TestLoadContextRejectsByteLimits(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "context.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"what_changed":"`+strings.Repeat("x", maxContextFieldBytes)+`a"}`), 0o600))
	_, err := LoadContext(path)
	require.ErrorContains(t, err, "what_changed")

	require.NoError(t, os.WriteFile(path, bytes.Repeat([]byte("x"), maxContextBytes+1), 0o600))
	_, err = LoadContext(path)
	require.ErrorContains(t, err, "exceeds")
}

func TestSessionPersistsImplementationContextWithoutChangingSnapshot(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.Black, color.White)
	context := ImplementationContext{
		Title:           "Spacing review",
		WhatChanged:     "Moved the button.\nKept the snapshot fixed.",
		WhatToTest:      "Try <script> as text.",
		ExpectedOutcome: "Button aligns with the panel.",
		Limitations:     "Desktop only.",
		SourceReference: "commit:abc123",
	}
	session, err := NewSessionWithContext(reference, actual, filepath.Join(dir, "rounds"), "", 0, 0.1, context)
	require.NoError(t, err)
	result, err := session.Submit(FeedbackRequest{Decision: decisionApproved})
	require.NoError(t, err)
	require.Equal(t, context, result.Context)
	feedback := mustReadFeedback(t, result.FeedbackPath)
	require.Equal(t, context, feedback.Context)
	manifest := mustRead(t, filepath.Join(session.Root(), snapshotManifestName))
	var persisted struct {
		Context ImplementationContext `json:"context"`
	}
	require.NoError(t, json.Unmarshal(manifest, &persisted))
	require.Equal(t, context, persisted.Context)
	require.Equal(t, session.Snapshot(), result.Snapshot)
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

func TestHandlerServesEmbeddedFrontendAssets(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.Black, color.White)
	session, err := NewSession(reference, actual, dir, "", 0, 0.1)
	require.NoError(t, err)
	server := httptest.NewServer(sessionHandler(session))
	defer server.Close()

	page, err := http.Get(server.URL + "/")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, page.StatusCode)
	pageBody, err := io.ReadAll(page.Body)
	require.NoError(t, err)
	require.NoError(t, page.Body.Close())
	require.Contains(t, string(pageBody), "/assets/app.js")

	for _, asset := range []struct {
		path        string
		contentType string
	}{
		{path: "/assets/style.css", contentType: "text/css; charset=utf-8"},
		{path: "/assets/app.js", contentType: "text/javascript; charset=utf-8"},
	} {
		response, getErr := http.Get(server.URL + asset.path)
		require.NoError(t, getErr)
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Equal(t, asset.contentType, response.Header.Get("Content-Type"))
		require.NoError(t, response.Body.Close())
	}
}

func TestHandlerExposesImplementationContext(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.Black, color.White)
	context := ImplementationContext{Title: "Context title", WhatToTest: "Check the popup."}
	session, err := NewSessionWithContext(reference, actual, dir, "", 0, 0.1, context)
	require.NoError(t, err)
	server := httptest.NewServer(sessionHandler(session))
	defer server.Close()

	response, err := http.Get(server.URL + "/api/session")
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()
	require.Equal(t, http.StatusOK, response.StatusCode)
	var payload struct {
		Context ImplementationContext `json:"context"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&payload))
	require.Equal(t, context, payload.Context)
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

	body := `{"decision":"submitted","notes":"fix the edge","annotations":[{"image":"actual","type":"point","x":2,"y":1,"note":"edge"},{"image":"reference","type":"rectangle","x":1,"y":0,"width":2,"height":2}]}`
	response, err = http.Post(server.URL+"/api/feedback", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var result Result
	require.NoError(t, json.NewDecoder(response.Body).Decode(&result))
	require.NoError(t, response.Body.Close())
	feedback := mustReadFeedback(t, result.FeedbackPath)
	require.Equal(t, "submitted", feedback.Decision)
	require.Equal(t, "note-0001", feedback.Annotations[0].ID)
	require.Equal(t, 2, feedback.Annotations[0].X)
	require.Equal(t, 1, feedback.Annotations[0].Y)
	require.Equal(t, 2, feedback.Annotations[1].Width)

	response, err = http.Post(server.URL+"/api/feedback", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.NoError(t, response.Body.Close())
}

func TestFeedbackPreservesEachSourceViewAndPixelGeometry(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.Black, color.White)
	session, err := NewSession(reference, actual, dir, "", 0, 0.1)
	require.NoError(t, err)

	result, err := session.Submit(FeedbackRequest{
		Decision: "submitted",
		Annotations: []Annotation{
			{Image: "reference", Type: "point", X: 0, Y: 1},
			{Image: "actual", Type: "rectangle", X: 1, Y: 0, Width: 2, Height: 2},
			{Image: "overlay", Type: "point", X: 2, Y: 1},
		},
	})
	require.NoError(t, err)

	feedback := mustReadFeedback(t, result.FeedbackPath)
	require.Equal(t, []string{"reference", "actual", "overlay"}, []string{
		feedback.Annotations[0].Image,
		feedback.Annotations[1].Image,
		feedback.Annotations[2].Image,
	})
	require.Equal(t, Annotation{ID: "note-0002", Image: "actual", Type: "rectangle", X: 1, Y: 0, Width: 2, Height: 2}, feedback.Annotations[1])
}

func TestHandlerRejectsInvalidDecisionPayloads(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.Black, color.White)
	session, err := NewSession(reference, actual, dir, "", 0, 0.1)
	require.NoError(t, err)
	server := httptest.NewServer(sessionHandler(session))
	defer server.Close()
	for _, body := range []string{
		`{"decision":"submitted"}`,
		`{"decision":"approved","notes":"not allowed"}`,
		`{"decision":"approved","annotations":[{"image":"actual","type":"point","x":0,"y":0}]}`,
	} {
		response, postErr := http.Post(server.URL+"/api/feedback", "application/json", bytes.NewBufferString(body))
		require.NoError(t, postErr)
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
		require.NoError(t, response.Body.Close())
	}
}

func TestHandlerRejectsUnknownAnnotationSource(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.Black, color.White)
	session, err := NewSession(reference, actual, dir, "", 0, 0.1)
	require.NoError(t, err)
	server := httptest.NewServer(sessionHandler(session))
	defer server.Close()

	response, err := http.Post(
		server.URL+"/api/feedback",
		"application/json",
		bytes.NewBufferString(`{"decision":"submitted","annotations":[{"image":"current","type":"point","x":0,"y":0}]}`),
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
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
	firstContext := ImplementationContext{Title: "First round", WhatToTest: "Check spacing."}
	first, err := NewSessionWithContext(reference, actual, dir, "", 0, 0.1, firstContext)
	require.NoError(t, err)
	firstResult, err := first.Submit(FeedbackRequest{Decision: "submitted", Notes: "fix it"})
	require.NoError(t, err)
	firstBytes, err := os.ReadFile(firstResult.FeedbackPath)
	require.NoError(t, err)

	secondContext := ImplementationContext{Title: "Second round", WhatToTest: "Check the new button."}
	second, err := NewSessionWithContext(reference, actual, dir, firstResult.FeedbackPath, 0, 0.1, secondContext)
	require.NoError(t, err)
	secondResult, err := second.Submit(FeedbackRequest{Decision: "approved"})
	require.NoError(t, err)
	require.Equal(t, 2, secondResult.Round)
	require.NotEqual(t, firstResult.FeedbackPath, secondResult.FeedbackPath)
	require.Equal(t, firstBytes, mustRead(t, firstResult.FeedbackPath))
	feedback := mustReadFeedback(t, secondResult.FeedbackPath)
	require.Equal(t, secondContext, feedback.Context)
	require.Equal(t, filepath.Join(second.Root(), "previous-feedback.json"), feedback.PreviousFeedback)
	require.Equal(t, firstContext, mustReadFeedback(t, feedback.PreviousFeedback).Context)
}

func TestServerLifecycleIsLoopbackOnly(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.Black, color.White)
	session, err := NewSession(reference, actual, dir, "", 0, 0.1)
	require.NoError(t, err)
	server := NewServer(session)
	url, err := server.Start()
	require.NoError(t, err)
	defer func() { _ = server.Close() }()
	require.Contains(t, url, "http://127.0.0.1:")
	response, err := http.Post(url+"/api/feedback", "application/json", bytes.NewBufferString(`{"decision":"approved"}`))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, response.Body.Close())
	result, waitErr := server.Wait()
	require.NoError(t, waitErr)
	require.Equal(t, "approved", result.Decision)
}

func TestServerWaitContextCancelsWithoutWritingFeedback(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.Black, color.White)
	session, err := NewSession(reference, actual, dir, "", 0, 0.1)
	require.NoError(t, err)
	server := NewServer(session)
	_, err = server.Start()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, waitErr := server.WaitContext(ctx)
	require.Error(t, waitErr)
	require.Contains(t, waitErr.Error(), "stopped before a decision")
	require.Empty(t, result.Decision)
	_, statErr := os.Stat(filepath.Join(session.Root(), "feedback.json"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestServerCloseBeforeDecisionReturnsExplicitError(t *testing.T) {
	dir := t.TempDir()
	reference, actual := writePNG(t, dir, "reference.png", color.Black, color.White)
	session, err := NewSession(reference, actual, dir, "", 0, 0.1)
	require.NoError(t, err)
	server := NewServer(session)
	_, err = server.Start()
	require.NoError(t, err)
	require.NoError(t, server.Close())
	result, waitErr := server.Wait()
	require.Error(t, waitErr)
	require.Contains(t, waitErr.Error(), "closed before a decision")
	require.Empty(t, result.Decision)
	_, statErr := os.Stat(filepath.Join(session.Root(), "feedback.json"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
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
