package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpkit "github.com/likearthian/apikit/v2/transport/http"
)

type stubRecorder struct {
	lastMethod   string
	lastPath     string
	lastCode     int
	lastDuration time.Duration
	lastSize     int64
	called       bool
}

func (s *stubRecorder) ObserveHTTPRequest(method, path string, code int, duration time.Duration, size int64) {
	s.lastMethod = method
	s.lastPath = path
	s.lastCode = code
	s.lastDuration = duration
	s.lastSize = size
	s.called = true
}

func TestServerFinalizer(t *testing.T) {
	rec := &stubRecorder{}

	ctx := context.Background()
	ctx = StartTimer(ctx, httptest.NewRequest(http.MethodGet, "/test", nil))
	ctx = context.WithValue(ctx, httpkit.ContextKeyRequestMethod, "GET")
	ctx = context.WithValue(ctx, httpkit.ContextKeyRequestPath, "/test")
	ctx = context.WithValue(ctx, httpkit.ContextKeyResponseSize, int64(42))

	finalizer := ServerFinalizer(rec)
	finalizer(ctx, http.StatusOK, httptest.NewRequest(http.MethodGet, "/test", nil))

	if !rec.called {
		t.Fatal("expected recorder to be called")
	}
	if rec.lastMethod != "GET" {
		t.Errorf("expected method GET, got %s", rec.lastMethod)
	}
	if rec.lastPath != "/test" {
		t.Errorf("expected path /test, got %s", rec.lastPath)
	}
	if rec.lastCode != http.StatusOK {
		t.Errorf("expected code 200, got %d", rec.lastCode)
	}
	if rec.lastSize != 42 {
		t.Errorf("expected size 42, got %d", rec.lastSize)
	}
	if rec.lastDuration <= 0 {
		t.Errorf("expected positive duration, got %v", rec.lastDuration)
	}
}

func TestServerFinalizerSkipsWithoutStartTime(t *testing.T) {
	rec := &stubRecorder{}
	ctx := context.Background()
	ctx = context.WithValue(ctx, httpkit.ContextKeyRequestMethod, "GET")

	finalizer := ServerFinalizer(rec)
	finalizer(ctx, http.StatusOK, httptest.NewRequest(http.MethodGet, "/test", nil))

	if rec.called {
		t.Fatal("expected recorder to not be called without start time")
	}
}
