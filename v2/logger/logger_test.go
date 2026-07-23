package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	apexlog "github.com/apex/log"
	"github.com/apex/log/handlers/memory"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
)

func TestAdapterSetLevel(t *testing.T) {
	t.Run("logrus", func(t *testing.T) {
		var output bytes.Buffer
		base := logrus.New()
		base.SetOutput(&output)
		base.SetFormatter(&logrus.JSONFormatter{DisableTimestamp: true})
		adapter := NewRusLog(base)

		adapter.SetLevel(ErrorLevel)
		adapter.Warn("ignored")
		adapter.Error("allowed")

		entries := decodeJSONLogs(t, output.Bytes())
		if len(entries) != 1 {
			t.Fatalf("entry count = %d, want 1", len(entries))
		}
		if entries[0]["msg"] != "allowed" {
			t.Errorf("message = %q, want %q", entries[0]["msg"], "allowed")
		}
		if entries[0]["level"] != "error" {
			t.Errorf("level = %q, want %q", entries[0]["level"], "error")
		}
	})

	t.Run("zerolog", func(t *testing.T) {
		var output bytes.Buffer
		adapter := NewZerolog(zerolog.New(&output))

		adapter.SetLevel(ErrorLevel)
		adapter.Warn("ignored")
		adapter.Error("allowed")

		entries := decodeJSONLogs(t, output.Bytes())
		if len(entries) != 1 {
			t.Fatalf("entry count = %d, want 1", len(entries))
		}
		if entries[0]["message"] != "allowed" {
			t.Errorf("message = %q, want %q", entries[0]["message"], "allowed")
		}
		if entries[0]["level"] != "error" {
			t.Errorf("level = %q, want %q", entries[0]["level"], "error")
		}
	})

	t.Run("apex", func(t *testing.T) {
		handler := memory.New()
		base := &apexlog.Logger{Handler: handler}
		adapter := NewApexLogger(base)

		adapter.SetLevel(ErrorLevel)
		adapter.Warn("ignored")
		adapter.Error("allowed")

		if len(handler.Entries) != 1 {
			t.Fatalf("entry count = %d, want 1", len(handler.Entries))
		}
		if handler.Entries[0].Message != "allowed" {
			t.Errorf("message = %q, want %q", handler.Entries[0].Message, "allowed")
		}
		if handler.Entries[0].Level != apexlog.ErrorLevel {
			t.Errorf("level = %s, want %s", handler.Entries[0].Level, apexlog.ErrorLevel)
		}
	})

	t.Run("noop", func(t *testing.T) {
		adapter := NewNoopLogger()

		adapter.SetLevel(ErrorLevel)
		adapter.Info("info", "request_id", "req-1")
		adapter.Debug("debug", "request_id", "req-1")
		adapter.Warn("warn", "request_id", "req-1")
		adapter.Error("error", "request_id", "req-1")
	})
}

func TestAdaptersPreserveMessageLevelAndFields(t *testing.T) {
	t.Run("logrus", func(t *testing.T) {
		var output bytes.Buffer
		base := logrus.New()
		base.SetOutput(&output)
		base.SetFormatter(&logrus.JSONFormatter{DisableTimestamp: true})

		NewRusLog(base).Info("served", "request_id", "req-1")

		entry := decodeJSONLog(t, output.Bytes())
		if entry["msg"] != "served" {
			t.Errorf("message = %q, want %q", entry["msg"], "served")
		}
		if entry["level"] != "info" {
			t.Errorf("level = %q, want %q", entry["level"], "info")
		}
		if entry["request_id"] != "req-1" {
			t.Errorf("request_id = %q, want %q", entry["request_id"], "req-1")
		}
	})

	t.Run("zerolog", func(t *testing.T) {
		var output bytes.Buffer

		NewZerolog(zerolog.New(&output)).Warn("served", "request_id", "req-1")

		entry := decodeJSONLog(t, output.Bytes())
		if entry["message"] != "served" {
			t.Errorf("message = %q, want %q", entry["message"], "served")
		}
		if entry["level"] != "warn" {
			t.Errorf("level = %q, want %q", entry["level"], "warn")
		}
		if entry["request_id"] != "req-1" {
			t.Errorf("request_id = %q, want %q", entry["request_id"], "req-1")
		}
	})

	t.Run("apex", func(t *testing.T) {
		handler := memory.New()
		base := &apexlog.Logger{Handler: handler}

		NewApexLogger(base).Error("served", "request_id", "req-1")

		if len(handler.Entries) != 1 {
			t.Fatalf("entry count = %d, want 1", len(handler.Entries))
		}
		entry := handler.Entries[0]
		if entry.Message != "served" {
			t.Errorf("message = %q, want %q", entry.Message, "served")
		}
		if entry.Level != apexlog.ErrorLevel {
			t.Errorf("level = %s, want %s", entry.Level, apexlog.ErrorLevel)
		}
		if entry.Fields["request_id"] != "req-1" {
			t.Errorf("request_id = %q, want %q", entry.Fields["request_id"], "req-1")
		}
	})
}

func decodeJSONLog(t *testing.T, data []byte) map[string]interface{} {
	t.Helper()

	entries := decodeJSONLogs(t, data)
	if len(entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(entries))
	}
	return entries[0]
}

func decodeJSONLogs(t *testing.T, data []byte) []map[string]interface{} {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(data))
	var entries []map[string]interface{}
	for {
		var entry map[string]interface{}
		err := decoder.Decode(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode JSON log: %v", err)
		}
		entries = append(entries, entry)
	}
	return entries
}
