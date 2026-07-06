package main

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/adnvilla/go-examples/examples/grpc/greeterpb"
)

// newTestClient starts the service on an in-memory bufconn listener — the
// gRPC equivalent of httptest: full client/server stack, no TCP, no ports.
func newTestClient(t *testing.T) greeterpb.GreeterClient {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	greeterpb.RegisterGreeterServer(server, greeterServer{})
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Errorf("serving: %v", err)
		}
	}()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(t.Context())
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("creating client: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return greeterpb.NewGreeterClient(conn)
}

func TestGreet(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)

	reply, err := client.Greet(t.Context(), &greeterpb.GreetRequest{Name: "Ada"})
	if err != nil {
		t.Fatalf("Greet: %v", err)
	}
	if got, want := reply.GetMessage(), "Hello, Ada!"; got != want {
		t.Errorf("Greet message = %q, want %q", got, want)
	}
}

func TestGreetRejectsEmptyName(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)

	_, err := client.Greet(t.Context(), &greeterpb.GreetRequest{})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("Greet with empty name: status = %v, want code %s", err, codes.InvalidArgument)
	}
}

func TestCountdownStreamsAllValues(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)

	stream, err := client.Countdown(t.Context(), &greeterpb.CountdownRequest{From: 3})
	if err != nil {
		t.Fatalf("Countdown: %v", err)
	}

	var got []int32
	for {
		msg, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		got = append(got, msg.GetValue())
	}

	want := []int32{3, 2, 1, 0}
	if len(got) != len(want) {
		t.Fatalf("received %d values (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("value[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}
