package health

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

type State struct {
	ready atomic.Bool
}

func NewState() *State {
	return &State{}
}

func (s *State) SetReady(ready bool) {
	s.ready.Store(ready)
}

func (s *State) LivenessHandler(w http.ResponseWriter, _ *http.Request) {
	writeStatus(w, http.StatusOK, "alive")
}

func (s *State) ReadinessHandler(w http.ResponseWriter, _ *http.Request) {
	if !s.ready.Load() {
		writeStatus(w, http.StatusServiceUnavailable, "not_ready")
		return
	}
	writeStatus(w, http.StatusOK, "ready")
}

func writeStatus(w http.ResponseWriter, code int, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
}
