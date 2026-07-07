// Demonstrates WebSockets in Go with coder/websocket: an HTTP endpoint that
// upgrades connections, a hub that broadcasts every message to all clients,
// per-client reader goroutines (the shape real clients need — control frames
// like pongs are only processed while a Read is in flight), ping for
// liveness, and clean closes with status codes. Server and two clients run
// in one process, so `go run .` shows a full chat round trip and terminates
// on its own.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"
)

// client pairs a connection with a continuously running reader goroutine.
// The pump is not an optimization: pings, pongs, and close frames are only
// processed while a Read is in flight, so a client that reads sporadically
// has a connection that is only sporadically alive.
type client struct {
	name string
	conn *websocket.Conn
	msgs chan string
}

func connect(ctx context.Context, name, url string) (*client, error) {
	conn, _, err := websocket.Dial(ctx, url, nil) //nolint:bodyclose // coder/websocket: resp.Body never needs closing
	if err != nil {
		return nil, fmt.Errorf("dialing %s: %w", url, err)
	}
	c := &client{name: name, conn: conn, msgs: make(chan string, 8)}
	go func() {
		defer close(c.msgs)
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				return // normal closure and errors both end the pump
			}
			c.msgs <- string(data)
		}
	}()
	return c, nil
}

// recv waits for the next broadcast to arrive at this client.
func (c *client) recv(ctx context.Context) (string, error) {
	select {
	case msg, ok := <-c.msgs:
		if !ok {
			return "", fmt.Errorf("%s: connection closed", c.name)
		}
		return msg, nil
	case <-ctx.Done():
		return "", fmt.Errorf("%s waiting for broadcast: %w", c.name, ctx.Err())
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Real HTTP server on a random loopback port; the hub is its handler.
	hub := NewHub()
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listening: %w", err)
	}
	server := &http.Server{Handler: hub, ReadHeaderTimeout: 5 * time.Second}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(lis) }()
	url := "ws://" + lis.Addr().String()
	fmt.Println("--- hub up; two clients connect ---")

	alice, err := connect(ctx, "alice", url)
	if err != nil {
		return err
	}
	defer alice.conn.CloseNow() //nolint:errcheck
	bob, err := connect(ctx, "bob", url)
	if err != nil {
		return err
	}
	defer bob.conn.CloseNow() //nolint:errcheck

	// Registration happens in the server's handler goroutines; wait until
	// the hub has seen both connections before broadcasting.
	if err := waitForClients(ctx, hub, 2); err != nil {
		return err
	}
	fmt.Printf("hub reports %d connected clients\n", hub.ClientCount())

	// Liveness: a ping round trip proves the peer is really there — TCP
	// being open says nothing. It works here precisely because alice's
	// reader pump is processing frames continuously.
	if err := alice.conn.Ping(ctx); err != nil {
		return fmt.Errorf("pinging: %w", err)
	}
	fmt.Println("alice: ping round trip ok")

	fmt.Println("\n--- broadcast: every message reaches every client ---")
	for _, msg := range []struct {
		sender *client
		text   string
	}{
		{alice, "alice: hola"},
		{bob, "bob: qué tal"},
	} {
		if err := msg.sender.conn.Write(ctx, websocket.MessageText, []byte(msg.text)); err != nil {
			return fmt.Errorf("%s sending: %w", msg.sender.name, err)
		}
		for _, receiver := range []*client{alice, bob} {
			got, err := receiver.recv(ctx)
			if err != nil {
				return err
			}
			fmt.Printf("%s received: %q\n", receiver.name, got)
		}
	}

	fmt.Println("\n--- clean close: status codes end the conversation ---")
	if err := alice.conn.Close(websocket.StatusNormalClosure, "done"); err != nil {
		return fmt.Errorf("closing alice: %w", err)
	}
	if err := bob.conn.Close(websocket.StatusNormalClosure, "done"); err != nil {
		return fmt.Errorf("closing bob: %w", err)
	}
	if err := waitForClients(ctx, hub, 0); err != nil {
		return err
	}
	fmt.Printf("hub reports %d connected clients after closes\n", hub.ClientCount())

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down server: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serving: %w", err)
	}
	fmt.Println("server: stopped cleanly")
	return nil
}

// waitForClients polls the hub until it has registered exactly n clients —
// registration runs in handler goroutines, so it's eventually consistent
// with the dials.
func waitForClients(ctx context.Context, hub *Hub, n int) error {
	for {
		if hub.ClientCount() == n {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for %d clients: %w", n, ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
}
