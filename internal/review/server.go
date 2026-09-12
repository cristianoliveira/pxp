package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

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
	s.httpServer = &http.Server{
		Handler:           s.handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	go func() { _ = s.httpServer.Serve(listener) }()
	return "http://" + listener.Addr().String(), nil
}

func (s *Server) Wait() (Result, error) {
	return s.WaitContext(context.Background())
}

// WaitContext waits for a decision or cancels the round safely. Cancellation
// closes the listener and records an operational error without writing feedback.
func (s *Server) WaitContext(ctx context.Context) (Result, error) {
	select {
	case completion := <-s.session.Completed():
		return completion.result, completion.err
	case <-ctx.Done():
		_ = s.Close()
		return Result{}, fmt.Errorf("review server stopped before a decision: %w", ctx.Err())
	}
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
	mux.HandleFunc("/assets/style.css", func(w http.ResponseWriter, r *http.Request) {
		s.handleFrontendAsset(w, r, "style.css", "text/css; charset=utf-8")
	})
	mux.HandleFunc("/assets/app.js", func(w http.ResponseWriter, r *http.Request) {
		s.handleFrontendAsset(w, r, "app.js", "text/javascript; charset=utf-8")
	})
	mux.HandleFunc("/image/reference.png", func(w http.ResponseWriter, r *http.Request) {
		s.handleImage(w, r, s.snapshot().Reference.Path)
	})
	mux.HandleFunc("/image/actual.png", func(w http.ResponseWriter, r *http.Request) {
		s.handleImage(w, r, s.snapshot().Actual.Path)
	})
	mux.HandleFunc("/image/overlay.png", func(w http.ResponseWriter, r *http.Request) {
		s.handleImage(w, r, s.snapshot().Overlay.Path)
	})
	mux.HandleFunc("/image/mask.png", func(w http.ResponseWriter, r *http.Request) {
		s.handleImage(w, r, s.snapshot().Mask.Path)
	})
	return securityHeaders(mux)
}

func (s *Server) snapshot() Snapshot { return s.session.snapshot }

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := frontendFile("index.html")
	if err != nil {
		http.Error(w, "review page unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) handleFrontendAsset(
	w http.ResponseWriter,
	r *http.Request,
	name, contentType string,
) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := frontendFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(data)
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]any{
		"version":    SchemaVersion,
		"session_id": s.snapshot().ID,
		"round":      s.session.round(),
		"snapshot":   s.snapshot(),
	})
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
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		next.ServeHTTP(w, r)
	})
}

const contentSecurityPolicy = "default-src 'self'; " +
	"img-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'"
