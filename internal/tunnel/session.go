package tunnel

import (
	"fmt"
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

// Session pairs a WebSocket connection with a TCP connection for bidirectional
// forwarding. The client and server packages use it to relay tunnel data.
type Session struct {
	tcp net.Conn
	ws  *websocket.Conn

	mu   sync.Mutex
	once sync.Once
	done chan struct{}
}

// NewSession creates a tunnel session from its WebSocket and TCP connections.
// It is called by the client and server packages after both connections are ready.
func NewSession(c *websocket.Conn, tcp net.Conn) *Session {
	return &Session{done: make(chan struct{}), ws: c, tcp: tcp}
}

// Close closes both connections exactly once and unblocks session readers.
func (s *Session) Close() {
	s.once.Do(func() {
		close(s.done)

		s.mu.Lock()
		defer s.mu.Unlock()

		if s.tcp != nil {
			_ = s.tcp.Close()
		}
		if s.ws != nil {
			_ = s.ws.Close()
		}
	})
}

// SendWS writes data as a binary WebSocket message.
func (s *Session) SendWS(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ws == nil {
		return fmt.Errorf("websocket is not connected")
	}

	return s.ws.WriteMessage(websocket.BinaryMessage, data)
}

// ReadWS reads the next WebSocket message.
func (s *Session) ReadWS() (messageType int, p []byte, err error) {
	return s.ws.ReadMessage()
}

// SendTCP writes data to the session's TCP connection.
func (s *Session) SendTCP(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tcp == nil {
		return fmt.Errorf("tcp connection is not connected")
	}

	_, err := s.tcp.Write(data)
	return err
}

// ReadTCP reads data from the session's TCP connection.
func (s *Session) ReadTCP(b []byte) (n int, err error) {
	return s.tcp.Read(b)
}
