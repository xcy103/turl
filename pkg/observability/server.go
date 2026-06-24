// Package observability provides an admin HTTP server that exposes operational
// endpoints — liveness, readiness, metrics, and profiling — on a port separate
// from the public API, so internal signals are never served to end users.
package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	"github.com/beihai0xff/turl/configs"
)

// readinessTimeout caps how long the readiness probe waits for dependencies.
const readinessTimeout = 3 * time.Second

// HealthCheck reports the health of a single dependency. A nil error means healthy.
type HealthCheck func(ctx context.Context) error

// Server is the admin/observability HTTP server.
type Server struct {
	httpServer *http.Server
	mux        *http.ServeMux

	mu     sync.RWMutex
	checks map[string]HealthCheck
}

// New builds an admin server bound to the configured address. The liveness,
// readiness, and pprof endpoints are registered immediately; a metrics handler
// can be attached later via SetMetricsHandler.
func New(c *configs.ObservabilityConfig) *Server {
	c = c.WithDefaults()

	mux := http.NewServeMux()
	s := &Server{
		mux:    mux,
		checks: make(map[string]HealthCheck),
		httpServer: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", c.AdminListen, c.AdminPort),
			Handler:           mux,
			ReadHeaderTimeout: readinessTimeout,
		},
	}

	mux.HandleFunc("/healthz", s.handleLiveness)
	mux.HandleFunc("/readyz", s.handleReadiness)
	registerPprof(mux)

	return s
}

// RegisterReadinessCheck adds a named dependency check to the readiness probe.
func (s *Server) RegisterReadinessCheck(name string, check HealthCheck) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks[name] = check
}

// SetMetricsHandler mounts a handler at /metrics (e.g. a Prometheus handler).
func (s *Server) SetMetricsHandler(h http.Handler) {
	s.mux.Handle("/metrics", h)
}

// Start runs the admin server. It blocks until the server stops; callers
// typically invoke it in a goroutine.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// HTTPServer exposes the underlying server so callers can shut it down gracefully.
func (s *Server) HTTPServer() *http.Server {
	return s.httpServer
}

// Addr returns the address the admin server listens on.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// handleLiveness reports that the process is up. It performs no dependency
// checks, matching Kubernetes liveness-probe semantics.
func (s *Server) handleLiveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// handleReadiness runs every registered check and reports 200 only when all
// pass, otherwise 503, with a per-dependency breakdown in the body.
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
	defer cancel()

	s.mu.RLock()
	checks := make(map[string]HealthCheck, len(s.checks))
	for name, check := range s.checks {
		checks[name] = check
	}
	s.mu.RUnlock()

	results := make(map[string]string, len(checks))
	ready := true

	for name, check := range checks {
		if err := check(ctx); err != nil {
			ready = false
			results[name] = err.Error()
		} else {
			results[name] = "ok"
		}
	}

	status := http.StatusOK
	overall := "ok"
	if !ready {
		status = http.StatusServiceUnavailable
		overall = "unavailable"
	}

	writeJSON(w, status, map[string]any{"status": overall, "checks": results})
}

// registerPprof wires the net/http/pprof handlers onto the given mux instead of
// the global DefaultServeMux, keeping profiling on the admin server only.
func registerPprof(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
