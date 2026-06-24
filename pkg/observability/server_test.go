package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/turl/configs"
)

func newTestServer() *Server {
	return New(&configs.ObservabilityConfig{Enable: true})
}

func TestServer_DefaultsApplied(t *testing.T) {
	s := New(&configs.ObservabilityConfig{Enable: true})
	assert.Equal(t, "0.0.0.0:9090", s.Addr())
}

func TestServer_Liveness(t *testing.T) {
	s := newTestServer()

	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}

func TestServer_Readiness_AllHealthy(t *testing.T) {
	s := newTestServer()
	s.RegisterReadinessCheck("db", func(context.Context) error { return nil })
	s.RegisterReadinessCheck("cache", func(context.Context) error { return nil })

	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"ok","checks":{"db":"ok","cache":"ok"}}`, w.Body.String())
}

func TestServer_Readiness_DependencyDown(t *testing.T) {
	s := newTestServer()
	s.RegisterReadinessCheck("db", func(context.Context) error { return nil })
	s.RegisterReadinessCheck("cache", func(context.Context) error { return errors.New("conn refused") })

	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.JSONEq(t, `{"status":"unavailable","checks":{"db":"ok","cache":"conn refused"}}`, w.Body.String())
}

func TestServer_Readiness_NoChecks(t *testing.T) {
	s := newTestServer()

	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	// With no registered checks the server is trivially ready.
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServer_SetMetricsHandler(t *testing.T) {
	s := newTestServer()
	s.SetMetricsHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("# metrics"))
	}))

	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "# metrics", w.Body.String())
}

func TestServer_PprofRegistered(t *testing.T) {
	s := newTestServer()

	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))

	assert.Equal(t, http.StatusOK, w.Code)
}
