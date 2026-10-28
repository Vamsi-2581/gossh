package tunnel

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// PingInterval is how often a WebSocket ping is sent. It is also frequent
	// enough to stop ngrok/Cloudflare from dropping an idle tunnel.
	PingInterval = 20 * time.Second
	// PongWait is how long the peer may stay silent before the session is dead.
	PongWait = 50 * time.Second
	// WriteWait bounds how long a ping write may take.
	WriteWait = 10 * time.Second
)

// Session pairs a WebSocket connection with a TCP connection for bidirectional
// forwarding. The client and server packages use it to relay tunnel data.
type Session struct {
	tcp net.Conn
	ws  *websocket.Conn

	wsMu  sync.Mutex // serializes WebSocket data writes
	tcpMu sync.Mutex // serializes TCP writes
	once  sync.Once
	done  chan struct{}
}

// NewSession creates a tunnel session and starts its keepalive.
func NewSession(c *websocket.Conn, tcp net.Conn) *Session {
	s := &Session{done: make(chan struct{}), ws: c, tcp: tcp}

	// Any pong (or data, see ReadWS) pushes the read deadline forward.
	// If nothing arrives for PongWait, ReadMessage fails and the session closes.
	_ = c.SetReadDeadline(time.Now().Add(PongWait))
	c.SetPongHandler(func(string) error {
		return c.SetReadDeadline(time.Now().Add(PongWait))
	})

	go s.keepalive()
	return s
}

// keepalive pings the peer until the session closes.
func (s *Session) keepalive() {
	t := time.NewTicker(PingInterval)
	defer t.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-t.C:
			// WriteControl is safe to call concurrently with other writers.
			err := s.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(WriteWait))
			if err != nil {
				s.Close()
				return
			}
		}
	}
}

// Close closes both connections exactly once and unblocks session readers
// and writers. It deliberately takes no write lock, so a stuck write
// cannot prevent the close.
func (s *Session) Close() {
	s.once.Do(func() {
		close(s.done)

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
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	if s.ws == nil {
		return fmt.Errorf("websocket is not connected")
	}

	return s.ws.WriteMessage(websocket.BinaryMessage, data)
}

// ReadWS reads the next WebSocket message. Receiving data also counts as
// proof the peer is alive.
func (s *Session) ReadWS() (messageType int, p []byte, err error) {
	messageType, p, err = s.ws.ReadMessage()
	if err == nil {
		_ = s.ws.SetReadDeadline(time.Now().Add(PongWait))
	}
	return messageType, p, err
}

// SendTCP writes data to the session's TCP connection.
func (s *Session) SendTCP(data []byte) error {
	s.tcpMu.Lock()
	defer s.tcpMu.Unlock()

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
