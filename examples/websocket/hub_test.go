package main

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// startHub serves the hub on an httptest server and returns its ws:// URL.
func startHub(t *testing.T) (*Hub, string) {
	t.Helper()
	hub := NewHub()
	ts := httptest.NewServer(hub)
	t.Cleanup(ts.Close)
	return hub, "ws" + strings.TrimPrefix(ts.URL, "http")
}

func TestBroadcastReachesAllClients(t *testing.T) {
	t.Parallel()
	hub, url := startHub(t)

	a, err := dial(t.Context(), url)
	if err != nil {
		t.Fatalf("dialing a: %v", err)
	}
	defer a.CloseNow() //nolint:errcheck
	b, err := dial(t.Context(), url)
	if err != nil {
		t.Fatalf("dialing b: %v", err)
	}
	defer b.CloseNow() //nolint:errcheck
	waitCount(t, hub, 2)

	if err := a.Write(t.Context(), websocket.MessageText, []byte("ping-all")); err != nil {
		t.Fatalf("writing: %v", err)
	}
	for name, conn := range map[string]*websocket.Conn{"a": a, "b": b} {
		_, data, err := conn.Read(t.Context())
		if err != nil {
			t.Fatalf("%s reading: %v", name, err)
		}
		if got := string(data); got != "ping-all" {
			t.Errorf("%s received %q, want %q", name, got, "ping-all")
		}
	}
}

func TestCloseUnregistersClient(t *testing.T) {
	t.Parallel()
	hub, url := startHub(t)

	conn, err := dial(t.Context(), url)
	if err != nil {
		t.Fatalf("dialing: %v", err)
	}
	waitCount(t, hub, 1)

	if err := conn.Close(websocket.StatusNormalClosure, "bye"); err != nil {
		t.Fatalf("closing: %v", err)
	}
	waitCount(t, hub, 0)
}

// waitCount polls until the hub registers exactly n clients; registration and
// unregistration happen in handler goroutines.
func waitCount(t *testing.T, hub *Hub, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for hub.ClientCount() != n {
		if time.Now().After(deadline) {
			t.Fatalf("hub has %d clients, want %d", hub.ClientCount(), n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
