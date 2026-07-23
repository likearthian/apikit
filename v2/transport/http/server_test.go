package http

import (
	"context"
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/likearthian/apikit/v2/api"
)

func TestNewServerRejectsMissingFunctions(t *testing.T) {
	endpoint := api.Endpoint[struct{}, struct{}](func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	})
	decoder := DecodeRequestFunc[struct{}](func(context.Context, *nethttp.Request) (struct{}, error) {
		return struct{}{}, nil
	})
	encoder := EncodeResponseFunc[struct{}](func(context.Context, nethttp.ResponseWriter, struct{}) error {
		return nil
	})

	tests := []struct {
		name     string
		endpoint api.Endpoint[struct{}, struct{}]
		decoder  DecodeRequestFunc[struct{}]
		encoder  EncodeResponseFunc[struct{}]
		wantErr  error
	}{
		{
			name:     "nil endpoint",
			endpoint: nil,
			decoder:  decoder,
			encoder:  encoder,
			wantErr:  ErrNilEndpoint,
		},
		{
			name:     "nil decoder",
			endpoint: endpoint,
			decoder:  nil,
			encoder:  encoder,
			wantErr:  ErrNilRequestDecoder,
		},
		{
			name:     "nil encoder",
			endpoint: endpoint,
			decoder:  decoder,
			encoder:  nil,
			wantErr:  ErrNilResponseEncoder,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, err := NewServer(test.endpoint, test.decoder, test.encoder)

			if server != nil {
				t.Error("NewServer() returned a non-nil server")
			}
			if !errors.Is(err, test.wantErr) {
				t.Errorf("NewServer() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestNewServerReturnsServerWithDefaults(t *testing.T) {
	server, err := NewServer(
		func(context.Context, struct{}) (struct{}, error) {
			return struct{}{}, errors.New("boom")
		},
		func(context.Context, *nethttp.Request) (struct{}, error) {
			return struct{}{}, nil
		},
		func(context.Context, nethttp.ResponseWriter, struct{}) error {
			return nil
		},
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if server == nil {
		t.Fatal("NewServer() returned a nil server")
	}
	if server.errorEncoder == nil {
		t.Error("NewServer() error encoder is nil")
	}
	if server.errorHandler == nil {
		t.Error("NewServer() error handler is nil")
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(nethttp.MethodGet, "/", nil)
	server.ServeHTTP(recorder, request)

	if recorder.Code != nethttp.StatusInternalServerError {
		t.Errorf("recorder status = %d, want %d", recorder.Code, nethttp.StatusInternalServerError)
	}
	if got := recorder.Body.String(); got != "boom" {
		t.Errorf("body = %q, want %q", got, "boom")
	}
}

func TestServerFinalizerObservesEncodedError(t *testing.T) {
	var finalizerCode int

	server, err := NewServer(
		func(context.Context, struct{}) (struct{}, error) {
			return struct{}{}, errors.New("boom")
		},
		func(context.Context, *nethttp.Request) (struct{}, error) {
			return struct{}{}, nil
		},
		func(context.Context, nethttp.ResponseWriter, struct{}) error {
			return nil
		},
		ServerErrorEncoder(func(_ context.Context, err error, w nethttp.ResponseWriter) {
			w.Header().Set("X-Error-Encoder", "custom")
			w.WriteHeader(nethttp.StatusTeapot)
			_, _ = w.Write([]byte("encoded: " + err.Error()))
		}),
		ServerFinalizer(func(_ context.Context, code int, _ *nethttp.Request) {
			finalizerCode = code
		}),
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(nethttp.MethodGet, "/", nil)
	server.ServeHTTP(recorder, request)

	if got := recorder.Header().Get("X-Error-Encoder"); got != "custom" {
		t.Errorf("X-Error-Encoder = %q, want %q", got, "custom")
	}
	if recorder.Code != nethttp.StatusTeapot {
		t.Errorf("recorder status = %d, want %d", recorder.Code, nethttp.StatusTeapot)
	}
	if got := recorder.Body.String(); got != "encoded: boom" {
		t.Errorf("body = %q, want %q", got, "encoded: boom")
	}
	if finalizerCode != nethttp.StatusTeapot {
		t.Errorf("finalizer status = %d, want %d", finalizerCode, nethttp.StatusTeapot)
	}
}

type contractError struct{}

func (contractError) Error() string {
	return "invalid input"
}

func (contractError) StatusCode() int {
	return nethttp.StatusUnprocessableEntity
}

func (contractError) Headers() nethttp.Header {
	return nethttp.Header{"X-Error-Code": []string{"invalid_input"}}
}

func TestDefaultErrorEncoderHonorsStatusHeadersAndBody(t *testing.T) {
	recorder := httptest.NewRecorder()

	DefaultErrorEncoder(context.Background(), contractError{}, recorder)

	if recorder.Code != nethttp.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, nethttp.StatusUnprocessableEntity)
	}
	if got := recorder.Header().Get("X-Error-Code"); got != "invalid_input" {
		t.Errorf("X-Error-Code = %q, want %q", got, "invalid_input")
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", got, "text/plain; charset=utf-8")
	}
	if got := recorder.Body.String(); got != "invalid input" {
		t.Errorf("body = %q, want %q", got, "invalid input")
	}
}
