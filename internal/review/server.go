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

const (
	listenNetwork = "tcp4"
	listenAddress = "127.0.0.1:0"
	urlScheme     = "http://"

	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 30 * time.Second

	indexRoute    = "/"
	sessionRoute  = "/api/session"
	feedbackRoute = "/api/feedback"
	styleAsset    = "style.css"
	scriptAsset   = "app.js"
	styleRoute    = "/assets/" + styleAsset
	scriptRoute   = "/assets/" + scriptAsset

	htmlContentType = "text/html; charset=utf-8"
	cssContentType  = "text/css; charset=utf-8"
	jsContentType   = "text/javascript; charset=utf-8"
	jsonContentType = "application/json"
	pngContentType  = "image/png"
	immutableCache  = "public, max-age=31536000, immutable"

	completedErrorFragment = "already completed"
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
	listener, err := net.Listen(listenNetwork, listenAddress)
	if err != nil {
		return "", fmt.Errorf("listen on localhost: %w", err)
	}
	s.listener = listener
	s.httpServer = &http.Server{
		Handler:           s.handler(),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	go func() { _ = s.httpServer.Serve(listener) }()
	return urlScheme + listener.Addr().String(), nil
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
	mux.HandleFunc(indexRoute, s.handleIndex)
	mux.HandleFunc(sessionRoute, s.handleSession)
	mux.HandleFunc(feedbackRoute, s.handleFeedback)
	mux.HandleFunc(styleRoute, func(w http.ResponseWriter, r *http.Request) {
		s.handleFrontendAsset(w, r, styleAsset, cssContentType)
	})
	mux.HandleFunc(scriptRoute, func(w http.ResponseWriter, r *http.Request) {
		s.handleFrontendAsset(w, r, scriptAsset, jsContentType)
	})
	mux.HandleFunc("/image/"+referenceFileName, func(w http.ResponseWriter, r *http.Request) {
		s.handleImage(w, r, s.snapshot().Reference.Path)
	})
	mux.HandleFunc("/image/"+actualFileName, func(w http.ResponseWriter, r *http.Request) {
		s.handleImage(w, r, s.snapshot().Actual.Path)
	})
	mux.HandleFunc("/image/"+overlayFileName, func(w http.ResponseWriter, r *http.Request) {
		s.handleImage(w, r, s.snapshot().Overlay.Path)
	})
	mux.HandleFunc("/image/"+maskFileName, func(w http.ResponseWriter, r *http.Request) {
		s.handleImage(w, r, s.snapshot().Mask.Path)
	})
	return securityHeaders(mux)
}

func (s *Server) snapshot() Snapshot { return s.session.snapshot }

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != indexRoute {
		http.NotFound(w, r)
		return
	}
	data, err := frontendFile("index.html")
	if err != nil {
		http.Error(w, "review page unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", htmlContentType)
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
	w.Header().Set("Cache-Control", immutableCache)
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
		"context":    s.session.Context(),
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
	w.Header().Set("Content-Type", pngContentType)
	w.Header().Set("Cache-Control", immutableCache)
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
		if strings.Contains(err.Error(), completedErrorFragment) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, result)
	s.session.complete(result)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", jsonContentType)
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
