package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scrape(t *testing.T) string {
	t.Helper()

	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, w.Code)

	body, err := io.ReadAll(w.Result().Body)
	require.NoError(t, err)

	return string(body)
}

func TestHandler_ExposesRegisteredMetrics(t *testing.T) {
	ObserveHTTPRequest(http.MethodGet, "/:short", "302", 0.012)
	RecordCacheResult(CacheLayerLocal, true)
	RecordCacheResult(CacheLayerDistributed, false)
	IncShortenCreated()

	body := scrape(t)

	assert.Contains(t, body, "turl_http_requests_total")
	assert.Contains(t, body, "turl_http_request_duration_seconds")
	assert.Contains(t, body, "turl_cache_requests_total")
	assert.Contains(t, body, "turl_shorten_created_total")
	// Go runtime collector is registered too.
	assert.Contains(t, body, "go_goroutines")
}

func TestObserveHTTPRequest_Labels(t *testing.T) {
	ObserveHTTPRequest(http.MethodPost, "/v1/management/shorten", "200", 0.05)

	body := scrape(t)

	assert.Contains(t, body,
		`turl_http_requests_total{method="POST",path="/v1/management/shorten",status="200"}`)
}

func TestRecordCacheResult_HitAndMiss(t *testing.T) {
	RecordCacheResult(CacheLayerLocal, true)
	RecordCacheResult(CacheLayerLocal, false)

	body := scrape(t)

	assert.True(t, strings.Contains(body, `turl_cache_requests_total{layer="local",result="hit"}`))
	assert.True(t, strings.Contains(body, `turl_cache_requests_total{layer="local",result="miss"}`))
}
