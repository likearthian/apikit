package transport

import (
	"context"
	"errors"
	"testing"
)

func TestNewNopErrorHandler(t *testing.T) {
	handler := NewNopErrorHandler()
	if handler == nil {
		t.Fatal("NewNopErrorHandler() returned nil")
	}

	handler.Handle(context.Background(), errors.New("ignored"))
}
