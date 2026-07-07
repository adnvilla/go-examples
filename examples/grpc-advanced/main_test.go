package main

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/adnvilla/go-examples/examples/grpc-advanced/echopb"
)

// newTestClient starts the full server (interceptors included) on an
// in-memory bufconn listener. withToken controls whether the client attaches
// the auth interceptors, so tests can exercise both sides of the check.
func newTestClient(t *testing.T, withToken bool) echopb.EchoClient {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(
		grpc.UnaryInterceptor(authUnaryInterceptor),
		grpc.StreamInterceptor(authStreamInterceptor),
	)
	echopb.RegisterEchoServer(server, echoServer{})
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Errorf("serving: %v", err)
		}
	}()
	t.Cleanup(server.Stop)

	opts := []grpc.DialOption{
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(t.Context())
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if withToken {
		opts = append(opts,
			grpc.WithUnaryInterceptor(tokenUnaryInterceptor),
			grpc.WithStreamInterceptor(tokenStreamInterceptor),
		)
	}

	conn, err := grpc.NewClient("passthrough:///bufnet", opts...)
	if err != nil {
		t.Fatalf("creating client: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return echopb.NewEchoClient(conn)
}

func TestUnaryEchoWithToken(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, true)

	reply, err := client.UnaryEcho(t.Context(), &echopb.EchoRequest{Message: "ping"})
	if err != nil {
		t.Fatalf("UnaryEcho: %v", err)
	}
	if got, want := reply.GetMessage(), "echo: ping"; got != want {
		t.Errorf("message = %q, want %q", got, want)
	}
}

func TestUnaryEchoRejectsMissingToken(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, false)

	_, err := client.UnaryEcho(t.Context(), &echopb.EchoRequest{Message: "ping"})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Fatalf("err = %v, want Unauthenticated", err)
	}
}

func TestStreamRejectsMissingToken(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, false)

	stream, err := client.BidiEcho(t.Context())
	if err != nil {
		t.Fatalf("opening stream: %v", err)
	}
	// Stream auth errors surface on the first Recv, not on open.
	_, err = stream.Recv()
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Fatalf("Recv err = %v, want Unauthenticated", err)
	}
}

func TestBidiEchoRoundTrip(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, true)

	stream, err := client.BidiEcho(t.Context())
	if err != nil {
		t.Fatalf("opening stream: %v", err)
	}

	for _, msg := range []string{"a", "b"} {
		if err := stream.Send(&echopb.EchoRequest{Message: msg}); err != nil {
			t.Fatalf("sending %q: %v", msg, err)
		}
		reply, err := stream.Recv()
		if err != nil {
			t.Fatalf("receiving echo of %q: %v", msg, err)
		}
		if got, want := reply.GetMessage(), "echo: "+msg; got != want {
			t.Errorf("echo of %q = %q, want %q", msg, got, want)
		}
	}

	if err := stream.CloseSend(); err != nil {
		t.Fatalf("CloseSend: %v", err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("Recv after CloseSend = %v, want io.EOF", err)
	}
}

func TestDeadlinePropagatesToHandler(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, true)

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.SlowEcho(ctx, &echopb.EchoRequest{Message: "slow"})
	elapsed := time.Since(start)

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.DeadlineExceeded {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
	if elapsed >= 5*time.Second {
		t.Fatalf("call took %v — the handler was awaited instead of canceled", elapsed)
	}
}
