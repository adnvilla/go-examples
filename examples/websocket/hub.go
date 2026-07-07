package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Hub is a broadcast fan-out: every message received from any client is
// written to all connected clients. This is the core of chat rooms, live
// dashboards, and collaborative editors.
type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

// NewHub returns an empty hub.
func NewHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]bool)}
}

// ClientCount reports how many connections are currently registered.
func (h *Hub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func (h *Hub) register(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
}

func (h *Hub) unregister(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
}

// broadcast writes msg to every registered client. Each write gets its own
// short deadline so one stuck client can't wedge the whole room — the
// slow-consumer problem every broadcast system has to answer.
func (h *Hub) broadcast(msg []byte) {
	h.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(h.clients))
	for c := range h.clients {
		conns = append(conns, c)
	}
	h.mu.Unlock()

	for _, conn := range conns {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		// A failed write means the client is gone or hopelessly slow; the
		// read loop will notice and unregister it. Dropping the message for
		// that client is the deliberate policy here.
		_ = conn.Write(ctx, websocket.MessageText, msg)
		cancel()
	}
}

// ServeHTTP upgrades the request to a WebSocket and runs the read loop:
// register, then read messages and broadcast each one until the client
// closes or errors, then unregister.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		// Accept has already written the HTTP error response.
		return
	}
	defer conn.CloseNow() //nolint:errcheck

	conn.SetReadLimit(1 << 20) // 1 MiB: bound what a single frame can cost
	h.register(conn)
	defer h.unregister(conn)

	for {
		_, data, err := conn.Read(r.Context())
		if err != nil {
			// Normal closure and network errors end the loop the same way;
			// websocket.CloseStatus(err) distinguishes them when needed.
			return
		}
		h.broadcast(data)
	}
}

// dial connects a client to the hub's ws:// URL.
func dial(ctx context.Context, url string) (*websocket.Conn, error) {
	conn, _, err := websocket.Dial(ctx, url, nil) //nolint:bodyclose // coder/websocket: resp.Body never needs closing
	if err != nil {
		return nil, fmt.Errorf("dialing %s: %w", url, err)
	}
	return conn, nil
}
