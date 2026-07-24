package prometheus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsObserveHTTPRequest(t *testing.T) {
	m := NewMetrics(prometheus.Labels{"service": "test"})

	m.ObserveHTTPRequest("GET", "/users", http.StatusOK, 10*time.Millisecond, 256)

	// Collect metrics and verify labels + values are present.
	reg := prometheus.NewRegistry()
	m.Register(reg)

	handler := Handler(reg)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, `http_request_duration_seconds`) {
		t.Error("expected http_request_duration_seconds in output")
	}
	if !strings.Contains(body, `http_requests_total`) {
		t.Error("expected http_requests_total in output")
	}
	if !strings.Contains(body, `http_response_size_bytes`) {
		t.Error("expected http_response_size_bytes in output")
	}
	if !strings.Contains(body, `service="test"`) {
		t.Error("expected service label in output")
	}
	if !strings.Contains(body, `method="GET"`) {
		t.Error("expected method label in output")
	}
}

func TestMetricsNilSafety(t *testing.T) {
	var m *Metrics
	// Must not panic.
	m.ObserveHTTPRequest("GET", "/", 200, 0, 0)
	m.Register(nil)
}

func TestMetricsNilLabels(t *testing.T) {
	m := NewMetrics(nil)
	if m == nil {
		t.Fatal("expected non-nil Metrics with nil labels")
	}
	m.ObserveHTTPRequest("POST", "/items", http.StatusCreated, 5*time.Millisecond, 128)
}

func TestHandler(t *testing.T) {
	reg := prometheus.NewRegistry()
	h := Handler(reg)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Errorf("expected text/plain content type, got %s", ct)
	}
}
