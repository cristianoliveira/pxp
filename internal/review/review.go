// Package review implements the short-lived, localhost-only annotated review round.
// It owns the review protocol and persistence; command wiring lives in commands.
package review

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cristianoliveira/pxp/internal/artifact"
	"github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/cristianoliveira/pxp/internal/imageio"
)

var contextFieldNames = map[string]struct{}{
	"title": {}, "what_changed": {}, "what_to_test": {},
	"expected_outcome": {}, "limitations": {}, "source_reference": {},
}

const (
	SchemaVersion = 1

	maxFeedbackBytes     = 64 * 1024
	maxContextBytes      = 16 * 1024
	maxContextFieldBytes = 4_000
	maxAnnotations       = 200
	maxGeneralNotes      = 10_000
	maxAnnotationNote    = 2_000

	firstRound             = 1
	snapshotIDHexSize      = 16
	jsonIndent             = "  "
	annotationIDFormat     = "note-%04d"
	reviewDirectoryPattern = "pxp-review-*"
	reviewRoundPattern     = "round-*"
	feedbackFileName       = "feedback.json"
	snapshotManifestName   = "snapshot.json"
	jsonFieldSeparator     = "\x00"

	reviewDirectoryMode = 0o700
	reviewFileMode      = 0o600
	immutableFileMode   = 0o444
	randomIDByteCount   = 8

	referenceFileName = "reference.png"
	actualFileName    = "actual.png"
	overlayFileName   = "overlay.png"
	maskFileName      = "mask.png"

	decisionSubmitted = "submitted"
	decisionApproved  = "approved"
	imageReference    = "reference"
	imageActual       = "actual"
	imageOverlay      = "overlay"
	annotationPoint   = "point"
	annotationRect    = "rectangle"
	contextFallback   = "No implementation context was provided for this round."
)

type Image struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Snapshot struct {
	ID        string `json:"id"`
	Reference Image  `json:"reference"`
	Actual    Image  `json:"actual"`
	Overlay   Image  `json:"overlay"`
	Mask      Image  `json:"mask"`
}

type ImplementationContext struct {
	Title           string `json:"title,omitempty"`
	WhatChanged     string `json:"what_changed,omitempty"`
	WhatToTest      string `json:"what_to_test,omitempty"`
	ExpectedOutcome string `json:"expected_outcome,omitempty"`
	Limitations     string `json:"limitations,omitempty"`
	SourceReference string `json:"source_reference,omitempty"`
}

type Annotation struct {
	ID     string `json:"id"`
	Image  string `json:"image"`
	Type   string `json:"type"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Note   string `json:"note,omitempty"`
}

type Feedback struct {
	Version          int                   `json:"version"`
	SessionID        string                `json:"session_id"`
	Round            int                   `json:"round"`
	Snapshot         Snapshot              `json:"snapshot"`
	Context          ImplementationContext `json:"context"`
	PreviousFeedback string                `json:"previous_feedback,omitempty"`
	Annotations      []Annotation          `json:"annotations,omitempty"`
	Notes            string                `json:"notes,omitempty"`
	Decision         string                `json:"decision"`
}

type FeedbackRequest struct {
	Annotations []Annotation `json:"annotations"`
	Notes       string       `json:"notes"`
	Decision    string       `json:"decision"`
}

type Result struct {
	Version      int                   `json:"version"`
	SessionID    string                `json:"session_id"`
	Round        int                   `json:"round"`
	Decision     string                `json:"decision"`
	FeedbackPath string                `json:"feedback_path"`
	Snapshot     Snapshot              `json:"snapshot"`
	Context      ImplementationContext `json:"context"`
}

type completion struct {
	result Result
	err    error
}

type Session struct {
	root           string
	feedback       string
	previous       string
	feedbackMu     sync.Mutex
	completionOnce sync.Once
	snapshot       Snapshot
	context        ImplementationContext
	completed      chan completion
	imageData      map[string][]byte
}

// NewSession creates a new immutable comparison snapshot. The input images are
// copied before comparison so later edits cannot change what the reviewer saw.
func NewSession(
	referencePath, actualPath, outputRoot, previousFeedback string,
	threshold uint8,
	perceptualThreshold float64,
) (*Session, error) {
	return NewSessionWithContext(
		referencePath,
		actualPath,
		outputRoot,
		previousFeedback,
		threshold,
		perceptualThreshold,
		ImplementationContext{},
	)
}

func NewSessionWithContext(
	referencePath, actualPath, outputRoot, previousFeedback string,
	threshold uint8,
	perceptualThreshold float64,
	context ImplementationContext,
) (*Session, error) {
	context, err := normalizeContext(context)
	if err != nil {
		return nil, err
	}
	if referencePath == "" || actualPath == "" {
		return nil, errors.New("reference and actual images are required")
	}
	if perceptualThreshold < 0 || math.IsNaN(perceptualThreshold) ||
		math.IsInf(perceptualThreshold, 0) {
		return nil, errors.New("perceptual threshold must be a finite non-negative number")
	}
	if outputRoot == "" {
		var err error
		outputRoot, err = os.MkdirTemp("", reviewDirectoryPattern)
		if err != nil {
			return nil, fmt.Errorf("create review directory: %w", err)
		}
	} else if err := os.MkdirAll(outputRoot, reviewDirectoryMode); err != nil {
		return nil, fmt.Errorf("create review output directory: %w", err)
	}
	root, err := os.MkdirTemp(outputRoot, reviewRoundPattern)
	if err != nil {
		return nil, fmt.Errorf("create review round: %w", err)
	}
	cleanupOnError := func() { _ = os.RemoveAll(root) }

	reference, err := copyImage(root, referenceFileName, referencePath)
	if err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("snapshot reference: %w", err)
	}
	actual, err := copyImage(root, actualFileName, actualPath)
	if err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("snapshot actual: %w", err)
	}
	if reference.Width != actual.Width || reference.Height != actual.Height {
		cleanupOnError()
		return nil, fmt.Errorf(
			"image dimensions differ: reference is %dx%d, actual is %dx%d",
			reference.Width,
			reference.Height,
			actual.Width,
			actual.Height,
		)
	}
	images, err := imageio.LoadDecodedImages(reference.Path, actual.Path)
	if err != nil {
		cleanupOnError()
		return nil, err
	}
	comparison, mask, err := images.Compare(threshold, perceptualThreshold, nil, nil)
	if err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("compare images: %w", err)
	}
	maskPath := filepath.Join(root, maskFileName)
	if err := imageio.WritePNG(maskPath, mask); err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("write review mask: %w", err)
	}
	overlayPath := filepath.Join(root, overlayFileName)
	overlay, err := images.Overlay(nil, nil)
	if err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("create review overlay: %w", err)
	}
	if err := imageio.WritePNG(overlayPath, overlay); err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("write review overlay: %w", err)
	}
	overlayInfo, err := immutableImage(
		root,
		overlayFileName,
		overlayPath,
		reference.Width,
		reference.Height,
	)
	if err != nil {
		cleanupOnError()
		return nil, err
	}
	maskInfo, err := immutableImage(root, maskFileName, maskPath, reference.Width, reference.Height)
	if err != nil {
		cleanupOnError()
		return nil, err
	}

	snapshotID := hashStrings(
		reference.SHA256,
		actual.SHA256,
		fmt.Sprintf("%dx%d", comparison.Width, comparison.Height),
	)[:snapshotIDHexSize]
	previousCopy := ""
	if previousFeedback != "" {
		previousCopy, err = copyPreviousFeedback(root, previousFeedback)
		if err != nil {
			cleanupOnError()
			return nil, err
		}
	}
	id, err := randomID("review")
	if err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("create review session id: %w", err)
	}
	round := firstRound
	if previousCopy != "" {
		var prior Feedback
		data, readErr := os.ReadFile(previousCopy)
		if readErr != nil || json.Unmarshal(data, &prior) != nil || !validFeedbackHeader(prior) {
			cleanupOnError()
			return nil, errors.New("previous feedback is not valid review feedback")
		}
		round = prior.Round + 1
	}
	snapshot := Snapshot{
		ID:        id + "-" + snapshotID,
		Reference: reference,
		Actual:    actual,
		Overlay:   overlayInfo,
		Mask:      maskInfo,
	}
	manifest := struct {
		Version    int                       `json:"version"`
		Round      int                       `json:"round"`
		Snapshot   Snapshot                  `json:"snapshot"`
		Context    ImplementationContext     `json:"context"`
		Comparison imagediff.ImageComparison `json:"comparison"`
	}{SchemaVersion, round, snapshot, context, comparison}
	manifestData, err := json.MarshalIndent(manifest, "", jsonIndent)
	if err != nil {
		cleanupOnError()
		return nil, err
	}
	manifestPath := filepath.Join(root, snapshotManifestName)
	if err := artifact.WriteFile(manifestPath, append(manifestData, '\n'), reviewFileMode); err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("write snapshot manifest: %w", err)
	}
	snapshotImages := []Image{
		snapshot.Reference,
		snapshot.Actual,
		snapshot.Overlay,
		snapshot.Mask,
	}
	imageData := make(map[string][]byte, len(snapshotImages))
	for _, image := range snapshotImages {
		data, readErr := os.ReadFile(image.Path)
		if readErr != nil {
			cleanupOnError()
			return nil, fmt.Errorf("read snapshot image: %w", readErr)
		}
		imageData[image.Path] = data
	}
	return &Session{
		root: root, feedback: filepath.Join(root, feedbackFileName), previous: previousCopy,
		snapshot: snapshot, context: context, completed: make(chan completion, 1), imageData: imageData,
	}, nil
}

func LoadContext(path string) (ImplementationContext, error) {
	if path == "" {
		return normalizeContext(ImplementationContext{})
	}
	file, err := os.Open(path)
	if err != nil {
		return ImplementationContext{}, fmt.Errorf("read context file: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxContextBytes+1))
	if err != nil {
		return ImplementationContext{}, fmt.Errorf("read context file: %w", err)
	}
	if len(data) > maxContextBytes {
		return ImplementationContext{}, fmt.Errorf("context file exceeds %d bytes", maxContextBytes)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return normalizeContext(ImplementationContext{})
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var fields map[string]json.RawMessage
	if err := decoder.Decode(&fields); err != nil {
		return ImplementationContext{}, fmt.Errorf("decode context file: %w", err)
	}
	if fields == nil {
		return ImplementationContext{}, errors.New("context file must contain one JSON object")
	}
	var context ImplementationContext
	for name, raw := range fields {
		if _, ok := contextFieldNames[name]; !ok {
			return ImplementationContext{}, fmt.Errorf("context file contains unknown field %q", name)
		}
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return ImplementationContext{}, fmt.Errorf("context field %s must be a string", name)
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return ImplementationContext{}, fmt.Errorf("context field %s must be a string: %w", name, err)
		}
		switch name {
		case "title":
			context.Title = value
		case "what_changed":
			context.WhatChanged = value
		case "what_to_test":
			context.WhatToTest = value
		case "expected_outcome":
			context.ExpectedOutcome = value
		case "limitations":
			context.Limitations = value
		case "source_reference":
			context.SourceReference = value
		}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ImplementationContext{}, errors.New("context file must contain one JSON object")
	}
	return normalizeContext(context)
}

func normalizeContext(context ImplementationContext) (ImplementationContext, error) {
	context.Title = strings.TrimSpace(context.Title)
	context.WhatChanged = strings.TrimSpace(context.WhatChanged)
	context.WhatToTest = strings.TrimSpace(context.WhatToTest)
	context.ExpectedOutcome = strings.TrimSpace(context.ExpectedOutcome)
	context.Limitations = strings.TrimSpace(context.Limitations)
	context.SourceReference = strings.TrimSpace(context.SourceReference)
	fields := []struct {
		name  string
		value string
	}{
		{"title", context.Title},
		{"what_changed", context.WhatChanged},
		{"what_to_test", context.WhatToTest},
		{"expected_outcome", context.ExpectedOutcome},
		{"limitations", context.Limitations},
		{"source_reference", context.SourceReference},
	}
	for _, field := range fields {
		if len(field.value) > maxContextFieldBytes {
			return ImplementationContext{}, fmt.Errorf("context field %s exceeds %d bytes", field.name, maxContextFieldBytes)
		}
	}
	if context.Title == "" && context.WhatChanged == "" && context.WhatToTest == "" &&
		context.ExpectedOutcome == "" && context.Limitations == "" && context.SourceReference == "" {
		context.Title = contextFallback
	}
	return context, nil
}

func (s *Session) Snapshot() Snapshot             { return s.snapshot }
func (s *Session) Context() ImplementationContext { return s.context }
func (s *Session) Root() string                   { return s.root }
func (s *Session) Completed() <-chan completion   { return s.completed }

func (s *Session) Submit(request FeedbackRequest) (Result, error) {
	s.feedbackMu.Lock()
	defer s.feedbackMu.Unlock()
	if _, err := os.Stat(s.feedback); err == nil {
		return Result{}, errors.New("review round already completed")
	}
	if err := validateRequest(
		request,
		s.snapshot.Reference.Width,
		s.snapshot.Reference.Height,
	); err != nil {
		return Result{}, err
	}
	annotations := make([]Annotation, len(request.Annotations))
	copy(annotations, request.Annotations)
	for index := range annotations {
		annotations[index].ID = fmt.Sprintf(annotationIDFormat, index+1)
	}
	feedback := Feedback{
		Version: SchemaVersion, SessionID: s.snapshot.ID, Round: s.round(), Snapshot: s.snapshot,
		Context: s.context, PreviousFeedback: s.previous,
		Annotations: annotations,
		Notes:       request.Notes,
		Decision:    request.Decision,
	}
	data, err := json.MarshalIndent(feedback, "", jsonIndent)
	if err != nil {
		return Result{}, err
	}
	file, err := os.OpenFile(s.feedback, os.O_WRONLY|os.O_CREATE|os.O_EXCL, reviewFileMode)
	if err != nil {
		return Result{}, fmt.Errorf("persist feedback: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return Result{}, fmt.Errorf("persist feedback: %w", err)
	}
	if err := file.Close(); err != nil {
		return Result{}, fmt.Errorf("persist feedback: %w", err)
	}
	result := Result{
		Version:      SchemaVersion,
		SessionID:    feedback.SessionID,
		Round:        feedback.Round,
		Decision:     feedback.Decision,
		FeedbackPath: s.feedback,
		Snapshot:     s.snapshot,
		Context:      s.context,
	}
	return result, nil
}

func (s *Session) complete(result Result) {
	s.completionOnce.Do(func() { s.completed <- completion{result: result} })
}

func (s *Session) abort(err error) {
	s.completionOnce.Do(func() { s.completed <- completion{err: err} })
}

func (s *Session) round() int {
	if s.previous == "" {
		return firstRound
	}
	data, err := os.ReadFile(s.previous)
	if err != nil {
		return firstRound
	}
	var prior Feedback
	if json.Unmarshal(data, &prior) != nil {
		return firstRound
	}
	return prior.Round + 1
}

func validateRequest(request FeedbackRequest, width, height int) error {
	if request.Decision != decisionSubmitted && request.Decision != decisionApproved {
		return errors.New("decision must be submitted or approved")
	}
	hasNotes := strings.TrimSpace(request.Notes) != ""
	if request.Decision == decisionSubmitted && !hasNotes && len(request.Annotations) == 0 {
		return errors.New("submitted feedback requires a general note or annotation")
	}
	if request.Decision == decisionApproved && (hasNotes || len(request.Annotations) > 0) {
		return errors.New("approval cannot include notes or annotations")
	}
	if len(request.Notes) > maxGeneralNotes {
		return fmt.Errorf("notes exceed %d characters", maxGeneralNotes)
	}
	if len(request.Annotations) > maxAnnotations {
		return fmt.Errorf("too many annotations (maximum %d)", maxAnnotations)
	}
	for index, annotation := range request.Annotations {
		if annotation.Image != imageReference &&
			annotation.Image != imageActual &&
			annotation.Image != imageOverlay {
			return fmt.Errorf("annotation %d has invalid image %q", index+1, annotation.Image)
		}
		if annotation.Type != annotationPoint && annotation.Type != annotationRect {
			return fmt.Errorf("annotation %d has invalid type %q", index+1, annotation.Type)
		}
		if annotation.X < 0 || annotation.Y < 0 || annotation.X >= width || annotation.Y >= height {
			return fmt.Errorf("annotation %d starts outside %dx%d image", index+1, width, height)
		}
		if annotation.Type == annotationPoint && (annotation.Width != 0 || annotation.Height != 0) {
			return fmt.Errorf("annotation %d point must not have dimensions", index+1)
		}
		outsideRectangle := annotation.Width <= 0 || annotation.Height <= 0 ||
			annotation.X+annotation.Width > width ||
			annotation.Y+annotation.Height > height
		if annotation.Type == annotationRect && outsideRectangle {
			return fmt.Errorf("annotation %d rectangle is outside %dx%d image", index+1, width, height)
		}
		if len(annotation.Note) > maxAnnotationNote {
			return fmt.Errorf("annotation %d note exceeds %d characters", index+1, maxAnnotationNote)
		}
	}
	return nil
}

func copyImage(root, name, source string) (Image, error) {
	width, height, err := imageio.PNGDimensions(source)
	if err != nil {
		return Image{}, err
	}
	target := filepath.Join(root, name)
	data, err := os.ReadFile(source)
	if err != nil {
		return Image{}, err
	}
	if err := os.WriteFile(target, data, reviewFileMode); err != nil {
		return Image{}, err
	}
	if err := os.Chmod(target, immutableFileMode); err != nil {
		return Image{}, err
	}
	return Image{
		Name:   strings.TrimSuffix(name, filepath.Ext(name)),
		Path:   target,
		SHA256: hashBytes(data),
		Width:  width,
		Height: height,
	}, nil
}

func immutableImage(root, name, source string, width, height int) (Image, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return Image{}, err
	}
	if err := os.Chmod(source, 0o444); err != nil {
		return Image{}, err
	}
	return Image{
		Name:   strings.TrimSuffix(name, filepath.Ext(name)),
		Path:   filepath.Join(root, name),
		SHA256: hashBytes(data),
		Width:  width,
		Height: height,
	}, nil
}

func copyPreviousFeedback(root, source string) (string, error) {
	info, err := os.Stat(source)
	if err != nil {
		return "", fmt.Errorf("read previous feedback: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("previous feedback must be a regular file")
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("read previous feedback: %w", err)
	}
	var prior Feedback
	if err := json.Unmarshal(data, &prior); err != nil || !validFeedbackHeader(prior) {
		return "", errors.New("previous feedback is not valid review feedback")
	}
	target := filepath.Join(root, "previous-feedback.json")
	if err := os.WriteFile(target, data, 0o444); err != nil {
		return "", fmt.Errorf("snapshot previous feedback: %w", err)
	}
	return target, nil
}

func validFeedbackHeader(feedback Feedback) bool {
	return feedback.Version == SchemaVersion &&
		feedback.Round > 0 &&
		feedback.SessionID != "" &&
		(feedback.Decision == "submitted" || feedback.Decision == "approved")
}

func hashBytes(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func hashStrings(values ...string) string {
	h := sha256.New()
	for _, value := range values {
		_, _ = io.WriteString(h, value+jsonFieldSeparator)
	}
	return hex.EncodeToString(h.Sum(nil))
}
func randomID(prefix string) (string, error) {
	var data [randomIDByteCount]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(data[:]), nil
}
