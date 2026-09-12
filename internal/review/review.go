// Package review implements the short-lived, localhost-only annotated review round.
// It owns the review protocol and persistence; command wiring lives in commands.
package review

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cristianoliveira/pxp/internal/artifact"
	"github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/cristianoliveira/pxp/internal/imageio"
)

const (
	SchemaVersion     = 1
	maxFeedbackBytes  = 64 * 1024
	maxAnnotations    = 200
	maxGeneralNotes   = 10_000
	maxAnnotationNote = 2_000
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
	Version          int          `json:"version"`
	SessionID        string       `json:"session_id"`
	Round            int          `json:"round"`
	Snapshot         Snapshot     `json:"snapshot"`
	PreviousFeedback string       `json:"previous_feedback,omitempty"`
	Annotations      []Annotation `json:"annotations,omitempty"`
	Notes            string       `json:"notes,omitempty"`
	Decision         string       `json:"decision"`
}

type FeedbackRequest struct {
	Annotations []Annotation `json:"annotations"`
	Notes       string       `json:"notes"`
	Decision    string       `json:"decision"`
}

type Result struct {
	Version      int      `json:"version"`
	SessionID    string   `json:"session_id"`
	Round        int      `json:"round"`
	Decision     string   `json:"decision"`
	FeedbackPath string   `json:"feedback_path"`
	Snapshot     Snapshot `json:"snapshot"`
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
	completed      chan completion
	imageData      map[string][]byte
}

// NewSession creates a new immutable comparison snapshot. The input images are
// copied before comparison so later edits cannot change what the reviewer saw.
func NewSession(referencePath, actualPath, outputRoot, previousFeedback string, threshold uint8, perceptualThreshold float64) (*Session, error) {
	if referencePath == "" || actualPath == "" {
		return nil, errors.New("reference and actual images are required")
	}
	if perceptualThreshold < 0 || math.IsNaN(perceptualThreshold) || math.IsInf(perceptualThreshold, 0) {
		return nil, errors.New("perceptual threshold must be a finite non-negative number")
	}
	if outputRoot == "" {
		var err error
		outputRoot, err = os.MkdirTemp("", "pxp-review-*")
		if err != nil {
			return nil, fmt.Errorf("create review directory: %w", err)
		}
	} else if err := os.MkdirAll(outputRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create review output directory: %w", err)
	}
	root, err := os.MkdirTemp(outputRoot, "round-*")
	if err != nil {
		return nil, fmt.Errorf("create review round: %w", err)
	}
	cleanupOnError := func() { _ = os.RemoveAll(root) }

	reference, err := copyImage(root, "reference.png", referencePath)
	if err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("snapshot reference: %w", err)
	}
	actual, err := copyImage(root, "actual.png", actualPath)
	if err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("snapshot actual: %w", err)
	}
	if reference.Width != actual.Width || reference.Height != actual.Height {
		cleanupOnError()
		return nil, fmt.Errorf("image dimensions differ: reference is %dx%d, actual is %dx%d", reference.Width, reference.Height, actual.Width, actual.Height)
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
	maskPath := filepath.Join(root, "mask.png")
	if err := imageio.WritePNG(maskPath, mask); err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("write review mask: %w", err)
	}
	overlayPath := filepath.Join(root, "overlay.png")
	overlay, err := images.Overlay(nil, nil)
	if err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("create review overlay: %w", err)
	}
	if err := imageio.WritePNG(overlayPath, overlay); err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("write review overlay: %w", err)
	}
	overlayInfo, err := immutableImage(root, "overlay.png", overlayPath, reference.Width, reference.Height)
	if err != nil {
		cleanupOnError()
		return nil, err
	}
	maskInfo, err := immutableImage(root, "mask.png", maskPath, reference.Width, reference.Height)
	if err != nil {
		cleanupOnError()
		return nil, err
	}

	snapshotID := hashStrings(reference.SHA256, actual.SHA256, fmt.Sprintf("%dx%d", comparison.Width, comparison.Height))[:16]
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
	round := 1
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
		Comparison imagediff.ImageComparison `json:"comparison"`
	}{SchemaVersion, round, snapshot, comparison}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		cleanupOnError()
		return nil, err
	}
	if err := artifact.WriteFile(filepath.Join(root, "snapshot.json"), append(manifestData, '\n'), 0o600); err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("write snapshot manifest: %w", err)
	}
	imageData := make(map[string][]byte, 4)
	for _, image := range []Image{snapshot.Reference, snapshot.Actual, snapshot.Overlay, snapshot.Mask} {
		data, readErr := os.ReadFile(image.Path)
		if readErr != nil {
			cleanupOnError()
			return nil, fmt.Errorf("read snapshot image: %w", readErr)
		}
		imageData[image.Path] = data
	}
	return &Session{
		root: root, feedback: filepath.Join(root, "feedback.json"), previous: previousCopy,
		snapshot: snapshot, completed: make(chan completion, 1), imageData: imageData,
	}, nil
}

func (s *Session) Snapshot() Snapshot           { return s.snapshot }
func (s *Session) Root() string                 { return s.root }
func (s *Session) Completed() <-chan completion { return s.completed }

func (s *Session) Submit(request FeedbackRequest) (Result, error) {
	s.feedbackMu.Lock()
	defer s.feedbackMu.Unlock()
	if _, err := os.Stat(s.feedback); err == nil {
		return Result{}, errors.New("review round already completed")
	}
	if err := validateRequest(request, s.snapshot.Reference.Width, s.snapshot.Reference.Height); err != nil {
		return Result{}, err
	}
	annotations := make([]Annotation, len(request.Annotations))
	copy(annotations, request.Annotations)
	for index := range annotations {
		annotations[index].ID = fmt.Sprintf("note-%04d", index+1)
	}
	feedback := Feedback{
		Version: SchemaVersion, SessionID: s.snapshot.ID, Round: s.round(), Snapshot: s.snapshot,
		PreviousFeedback: s.previous, Annotations: annotations, Notes: request.Notes, Decision: request.Decision,
	}
	data, err := json.MarshalIndent(feedback, "", "  ")
	if err != nil {
		return Result{}, err
	}
	file, err := os.OpenFile(s.feedback, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
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
	result := Result{Version: SchemaVersion, SessionID: feedback.SessionID, Round: feedback.Round, Decision: feedback.Decision, FeedbackPath: s.feedback, Snapshot: s.snapshot}
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
		return 1
	}
	data, err := os.ReadFile(s.previous)
	if err != nil {
		return 1
	}
	var prior Feedback
	if json.Unmarshal(data, &prior) != nil {
		return 1
	}
	return prior.Round + 1
}

func validateRequest(request FeedbackRequest, width, height int) error {
	if request.Decision != "submitted" && request.Decision != "approved" {
		return errors.New("decision must be submitted or approved")
	}
	hasNotes := strings.TrimSpace(request.Notes) != ""
	if request.Decision == "submitted" && !hasNotes && len(request.Annotations) == 0 {
		return errors.New("submitted feedback requires a general note or annotation")
	}
	if request.Decision == "approved" && (hasNotes || len(request.Annotations) > 0) {
		return errors.New("approval cannot include notes or annotations")
	}
	if len(request.Notes) > maxGeneralNotes {
		return fmt.Errorf("notes exceed %d characters", maxGeneralNotes)
	}
	if len(request.Annotations) > maxAnnotations {
		return fmt.Errorf("too many annotations (maximum %d)", maxAnnotations)
	}
	for index, annotation := range request.Annotations {
		if annotation.Image != "reference" && annotation.Image != "actual" && annotation.Image != "overlay" {
			return fmt.Errorf("annotation %d has invalid image %q", index+1, annotation.Image)
		}
		if annotation.Type != "point" && annotation.Type != "rectangle" {
			return fmt.Errorf("annotation %d has invalid type %q", index+1, annotation.Type)
		}
		if annotation.X < 0 || annotation.Y < 0 || annotation.X >= width || annotation.Y >= height {
			return fmt.Errorf("annotation %d starts outside %dx%d image", index+1, width, height)
		}
		if annotation.Type == "point" && (annotation.Width != 0 || annotation.Height != 0) {
			return fmt.Errorf("annotation %d point must not have dimensions", index+1)
		}
		if annotation.Type == "rectangle" && (annotation.Width <= 0 || annotation.Height <= 0 || annotation.X+annotation.Width > width || annotation.Y+annotation.Height > height) {
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
	if err := os.WriteFile(target, data, 0o600); err != nil {
		return Image{}, err
	}
	if err := os.Chmod(target, 0o444); err != nil {
		return Image{}, err
	}
	return Image{Name: strings.TrimSuffix(name, filepath.Ext(name)), Path: target, SHA256: hashBytes(data), Width: width, Height: height}, nil
}

func immutableImage(root, name, source string, width, height int) (Image, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return Image{}, err
	}
	if err := os.Chmod(source, 0o444); err != nil {
		return Image{}, err
	}
	return Image{Name: strings.TrimSuffix(name, filepath.Ext(name)), Path: filepath.Join(root, name), SHA256: hashBytes(data), Width: width, Height: height}, nil
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
	return feedback.Version == SchemaVersion && feedback.Round > 0 && feedback.SessionID != "" && (feedback.Decision == "submitted" || feedback.Decision == "approved")
}

func hashBytes(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func hashStrings(values ...string) string {
	h := sha256.New()
	for _, value := range values {
		_, _ = io.WriteString(h, value+"\x00")
	}
	return hex.EncodeToString(h.Sum(nil))
}
func randomID(prefix string) (string, error) {
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(data[:]), nil
}

// Server exposes one review round over a loopback listener only.
type Server struct {
	session    *Session
	listener   net.Listener
	httpServer *http.Server
	once       sync.Once
}

func NewServer(session *Session) *Server { return &Server{session: session} }
func (s *Server) Handler() http.Handler  { return s.handler() }
func (s *Server) Start() (string, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("listen on localhost: %w", err)
	}
	s.listener = listener
	s.httpServer = &http.Server{Handler: s.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second}
	go func() { _ = s.httpServer.Serve(listener) }()
	return "http://" + listener.Addr().String(), nil
}
func (s *Server) Wait() (Result, error) {
	completion := <-s.session.Completed()
	return completion.result, completion.err
}
func (s *Server) Close() error {
	s.once.Do(func() {
		if s.httpServer != nil {
			_ = s.httpServer.Close()
		} else if s.listener != nil {
			_ = s.listener.Close()
		}
		s.session.abort(errors.New("review server closed before a decision was recorded"))
	})
	return nil
}

func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/session", s.handleSession)
	mux.HandleFunc("/api/feedback", s.handleFeedback)
	mux.HandleFunc("/image/reference.png", func(w http.ResponseWriter, r *http.Request) { s.handleImage(w, r, s.snapshot().Reference.Path) })
	mux.HandleFunc("/image/actual.png", func(w http.ResponseWriter, r *http.Request) { s.handleImage(w, r, s.snapshot().Actual.Path) })
	mux.HandleFunc("/image/overlay.png", func(w http.ResponseWriter, r *http.Request) { s.handleImage(w, r, s.snapshot().Overlay.Path) })
	mux.HandleFunc("/image/mask.png", func(w http.ResponseWriter, r *http.Request) { s.handleImage(w, r, s.snapshot().Mask.Path) })
	return securityHeaders(mux)
}
func (s *Server) snapshot() Snapshot { return s.session.snapshot }
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, reviewHTML)
}
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]any{"version": SchemaVersion, "session_id": s.snapshot().ID, "round": s.session.round(), "snapshot": s.snapshot()})
}
func (s *Server) handleImage(w http.ResponseWriter, r *http.Request, path string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, ok := s.session.imageData[path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	_, _ = w.Write(data)
}
func (s *Server) handleFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxFeedbackBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request FeedbackRequest
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid feedback JSON", http.StatusBadRequest)
		return
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		http.Error(w, "feedback must contain one JSON object", http.StatusBadRequest)
		return
	}
	result, err := s.session.Submit(request)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "already completed") {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, result)
	s.session.complete(result)
}
func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}

const reviewHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>pxp visual review</title><style>body{font:16px system-ui,sans-serif;max-width:1200px;margin:2rem auto;padding:0 1rem;color:#202124} .images{display:grid;grid-template-columns:repeat(3,1fr);gap:1rem}.images img{max-width:100%;border:1px solid #bbb;background:#eee} canvas{display:block;max-width:100%;height:auto;border:2px solid #2563eb;cursor:crosshair}.controls{display:flex;flex-wrap:wrap;gap:.5rem;align-items:center;margin:1rem 0} textarea{width:100%;min-height:6rem} button{padding:.6rem 1rem} #status{min-height:1.5rem}.error{color:#b91c1c}.success{color:#166534} #annotations li{margin:.3rem 0}</style></head>
<body><h1>pxp visual review</h1><p>Click or drag on the actual image to add an annotation. Coordinates are stored in original pixels.</p><section class="images"><figure><figcaption>Reference</figcaption><img src="/image/reference.png" alt="Reference screenshot"></figure><figure><figcaption>Actual (annotate here)</figcaption><canvas id="canvas" aria-label="Actual screenshot annotation canvas"></canvas></figure><figure><figcaption>Overlay</figcaption><img src="/image/overlay.png" alt="Difference overlay"></figure></section>
<div class="controls"><label>Image <select id="image"><option value="actual">actual</option><option value="reference">reference</option><option value="overlay">overlay</option></select></label><label>Type <select id="type"><option value="point">point</option><option value="rectangle">rectangle</option></select></label><label>Annotation note <input id="annotation-note" maxlength="2000" size="40"></label></div><ol id="annotations"></ol><label for="notes">General notes</label><textarea id="notes" maxlength="10000"></textarea><div class="controls"><button id="submit" type="button">Submit feedback</button><button id="approve" type="button">Approve</button><span id="status" role="status"></span></div>
<script>
const canvas=document.getElementById('canvas'), ctx=canvas.getContext('2d'), image=document.getElementById('image'), type=document.getElementById('type'), note=document.getElementById('annotation-note'), list=document.getElementById('annotations'), status=document.getElementById('status'); const annotations=[]; let start=null;
const actual=new Image(); actual.onload=()=>{canvas.width=actual.naturalWidth;canvas.height=actual.naturalHeight;ctx.drawImage(actual,0,0);}; actual.src='/image/actual.png';
function point(event){const r=canvas.getBoundingClientRect(); return {x:Math.max(0,Math.min(canvas.width-1,Math.floor((event.clientX-r.left)*canvas.width/r.width))),y:Math.max(0,Math.min(canvas.height-1,Math.floor((event.clientY-r.top)*canvas.height/r.height)))};}
function redraw(){if(!actual.complete)return;ctx.drawImage(actual,0,0);ctx.strokeStyle='#ef4444';ctx.fillStyle='#ef4444';annotations.forEach(a=>{if(a.type==='point'){ctx.beginPath();ctx.arc(a.x,a.y,5,0,Math.PI*2);ctx.fill();}else{ctx.strokeRect(a.x,a.y,a.width,a.height);}});}
function refresh(){list.replaceChildren();annotations.forEach((a,i)=>{const li=document.createElement('li');li.textContent=(i+1)+'. '+a.image+' '+a.type+' @ '+a.x+','+a.y+(a.width?', '+a.width+'x'+a.height:'')+(a.note?' — '+a.note:'');list.appendChild(li);});redraw();}
canvas.addEventListener('pointerdown',e=>{start=point(e);canvas.setPointerCapture(e.pointerId);}); canvas.addEventListener('pointerup',e=>{if(!start)return;const end=point(e);let a={image:image.value,type:type.value,x:start.x,y:start.y,note:note.value};if(type.value==='rectangle'){a.x=Math.min(start.x,end.x);a.y=Math.min(start.y,end.y);a.width=Math.abs(end.x-start.x)+1;a.height=Math.abs(end.y-start.y)+1;if(a.width<2||a.height<2){start=null;return;}}annotations.push(a);start=null;note.value='';refresh();});
async function send(decision){const notes=document.getElementById('notes').value;if(decision==='approved'&&(annotations.length||notes.trim())){status.className='error';status.textContent='Remove notes and annotations before approving.';return;}if(decision==='submitted'&&!annotations.length&&!notes.trim()){status.className='error';status.textContent='Add a note or annotation before submitting.';return;}status.className='';status.textContent='Saving…';try{const response=await fetch('/api/feedback',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({annotations,notes,decision})});const data=await response.json();if(!response.ok)throw new Error(data.error||'feedback was rejected');status.className='success';status.textContent=decision==='approved'?'Approved. You may close this page.':'Feedback saved. You may close this page.';document.getElementById('submit').disabled=true;document.getElementById('approve').disabled=true;}catch(error){status.className='error';status.textContent=error.message;}}
document.getElementById('submit').onclick=()=>send('submitted');document.getElementById('approve').onclick=()=>send('approved');
</script></body></html>`
