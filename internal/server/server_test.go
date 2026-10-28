package server

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"gossh/internal/log"
	"gossh/internal/utils/test"
)

func newTestServer(maxConns int32) *GoSSHServer {
	return NewGoSSHServer(
		GoSSHServerConfiguration{MaxConns: int(maxConns), SSHPort: 1},
		log.NewLogger(log.ERROR, &test.MemLogger{}),
	)
}

// A request that never becomes a WebSocket must not keep its connection slot.
func TestHandleWSSessionReleasesSlotWhenUpgradeFails(t *testing.T) {
	server := newTestServer(3)

	recorder := httptest.NewRecorder()
	server.handleWSSession(recorder, httptest.NewRequest(http.MethodGet, "/ws", nil))

	if got := atomic.LoadInt32(&server.activeSessions); got != 0 {
		t.Fatalf("activeSessions = %d after a failed upgrade, want 0", got)
	}
}

// Requests over the limit are refused without taking a slot from the ones in use.
func TestHandleWSSessionRejectsOverLimit(t *testing.T) {
	server := newTestServer(2)
	atomic.StoreInt32(&server.activeSessions, 2)

	recorder := httptest.NewRecorder()
	server.handleWSSession(recorder, httptest.NewRequest(http.MethodGet, "/ws", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if got := atomic.LoadInt32(&server.activeSessions); got != 2 {
		t.Fatalf("activeSessions = %d after a refused request, want 2", got)
	}
}
