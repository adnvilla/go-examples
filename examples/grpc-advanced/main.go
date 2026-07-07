// Demonstrates the gRPC machinery production services rely on beyond plain
// RPCs: unary and stream interceptors on both sides (auth via metadata,
// logging), bidirectional streaming over one connection, and deadline
// propagation — the client's timeout cancels the handler's context on the
// server. Server and client run in one process so `go run .` is
// self-contained and deterministic.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/adnvilla/go-examples/examples/grpc-advanced/echopb"
)

const (
	authHeader = "authorization"
	// The hardcoded token exists to demonstrate the interceptor mechanics;
	// real services validate against an issuer (JWT/OAuth), never a constant.
	validToken = "Bearer demo-token" //nolint:gosec // intentional demo credential
)

// ── server ───────────────────────────────────────────────────────────────────

type echoServer struct {
	echopb.UnimplementedEchoServer
}

func (echoServer) UnaryEcho(_ context.Context, req *echopb.EchoRequest) (*echopb.EchoResponse, error) {
	return &echopb.EchoResponse{Message: "echo: " + req.GetMessage()}, nil
}

// SlowEcho simulates a 5s operation but selects on ctx.Done — when the
// client's 100ms deadline fires, the cancellation propagates here and the
// handler abandons the work instead of computing an answer nobody awaits.
func (echoServer) SlowEcho(ctx context.Context, req *echopb.EchoRequest) (*echopb.EchoResponse, error) {
	select {
	case <-time.After(5 * time.Second):
		return &echopb.EchoResponse{Message: "echo (slow): " + req.GetMessage()}, nil
	case <-ctx.Done():
		return nil, status.FromContextError(ctx.Err()).Err()
	}
}

// BidiEcho reads and writes on the same stream until the client closes its
// send side (io.EOF) — both directions are independent, which is the point
// of bidirectional streaming.
func (echoServer) BidiEcho(stream grpc.BidiStreamingServer[echopb.EchoRequest, echopb.EchoResponse]) error {
	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil // client finished sending; returning closes our side
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&echopb.EchoResponse{Message: "echo: " + req.GetMessage()}); err != nil {
			return err
		}
	}
}

// authUnaryInterceptor rejects unary calls without the expected bearer token.
// Interceptors are gRPC's middleware: cross-cutting concerns (auth, logging,
// metrics, retries) live here instead of inside every handler.
func authUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if err := checkAuth(ctx); err != nil {
		return nil, err
	}
	fmt.Printf("  [server interceptor] authorized unary %s\n", info.FullMethod)
	return handler(ctx, req)
}

// authStreamInterceptor is the streaming twin — note the separate registration:
// unary interceptors never see streaming RPCs and vice versa.
func authStreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := checkAuth(ss.Context()); err != nil {
		return err
	}
	fmt.Printf("  [server interceptor] authorized stream %s\n", info.FullMethod)
	return handler(srv, ss)
}

func checkAuth(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md.Get(authHeader)) == 0 || md.Get(authHeader)[0] != validToken {
		return status.Error(codes.Unauthenticated, "missing or invalid bearer token")
	}
	return nil
}

// ── client ───────────────────────────────────────────────────────────────────

// tokenUnaryInterceptor attaches the bearer token to every outgoing unary
// call — the client-side half of the auth story.
func tokenUnaryInterceptor(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	ctx = metadata.AppendToOutgoingContext(ctx, authHeader, validToken)
	return invoker(ctx, method, req, reply, cc, opts...)
}

// tokenStreamInterceptor does the same for streams.
func tokenStreamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, authHeader, validToken)
	return streamer(ctx, desc, cc, method, opts...)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listening: %w", err)
	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(authUnaryInterceptor),
		grpc.StreamInterceptor(authStreamInterceptor),
	)
	echopb.RegisterEchoServer(server, echoServer{})
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(lis) }()

	// One connection with the token interceptors, one deliberately without.
	authed, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(tokenUnaryInterceptor),
		grpc.WithStreamInterceptor(tokenStreamInterceptor),
	)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	defer authed.Close() //nolint:errcheck

	anonymous, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("creating anonymous client: %w", err)
	}
	defer anonymous.Close() //nolint:errcheck

	if err := demoAuth(ctx, echopb.NewEchoClient(authed), echopb.NewEchoClient(anonymous)); err != nil {
		return err
	}
	if err := demoDeadline(ctx, echopb.NewEchoClient(authed)); err != nil {
		return err
	}
	if err := demoBidi(ctx, echopb.NewEchoClient(authed)); err != nil {
		return err
	}

	server.GracefulStop()
	if err := <-serveErr; err != nil {
		return fmt.Errorf("serving: %w", err)
	}
	fmt.Println("\nserver: stopped cleanly")
	return nil
}

// demoAuth shows the interceptor pair doing its job: the connection whose
// interceptor attaches the token gets through; the anonymous one is rejected
// before the handler ever runs.
func demoAuth(ctx context.Context, authed, anonymous echopb.EchoClient) error {
	fmt.Println("--- interceptors: auth via metadata ---")

	reply, err := authed.UnaryEcho(ctx, &echopb.EchoRequest{Message: "hello"})
	if err != nil {
		return fmt.Errorf("authed UnaryEcho: %w", err)
	}
	fmt.Printf("with token:    %q\n", reply.GetMessage())

	_, err = anonymous.UnaryEcho(ctx, &echopb.EchoRequest{Message: "hello"})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		return fmt.Errorf("anonymous UnaryEcho: expected Unauthenticated, got %w", err)
	}
	fmt.Printf("without token: code=%s desc=%q (handler never ran)\n", st.Code(), st.Message())
	return nil
}

// demoDeadline shows cancellation crossing the wire: the client allows 100ms,
// the handler's 5s work is abandoned, and the client sees DeadlineExceeded
// promptly instead of after 5 seconds.
func demoDeadline(ctx context.Context, client echopb.EchoClient) error {
	fmt.Println("\n--- deadline propagation: client timeout cancels the handler ---")

	callCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.SlowEcho(callCtx, &echopb.EchoRequest{Message: "too slow"})
	elapsed := time.Since(start)

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.DeadlineExceeded {
		return fmt.Errorf("SlowEcho: expected DeadlineExceeded, got %w", err)
	}
	if elapsed >= 5*time.Second {
		return fmt.Errorf("deadline did not propagate: call took %v", elapsed)
	}
	fmt.Printf("code=%s after ~100ms (the 5s handler was canceled, not awaited)\n", st.Code())
	return nil
}

// demoBidi runs a bidirectional stream: three sends and three receives over
// one RPC, then CloseSend ends the conversation and Recv returns io.EOF.
func demoBidi(ctx context.Context, client echopb.EchoClient) error {
	fmt.Println("\n--- bidirectional streaming ---")

	stream, err := client.BidiEcho(ctx)
	if err != nil {
		return fmt.Errorf("opening bidi stream: %w", err)
	}

	for _, msg := range []string{"one", "two", "three"} {
		if err := stream.Send(&echopb.EchoRequest{Message: msg}); err != nil {
			return fmt.Errorf("sending %q: %w", msg, err)
		}
		reply, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("receiving echo of %q: %w", msg, err)
		}
		fmt.Printf("sent %q -> received %q\n", msg, reply.GetMessage())
	}

	if err := stream.CloseSend(); err != nil {
		return fmt.Errorf("closing send side: %w", err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("expected io.EOF after CloseSend, got %w", err)
	}
	fmt.Println("CloseSend acknowledged: server closed its side, Recv returned io.EOF")
	return nil
}
