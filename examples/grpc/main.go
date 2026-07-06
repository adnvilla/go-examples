// Demonstrates a gRPC service end to end — unary and server-streaming RPCs,
// status codes for errors — with server and client in one process so the
// example runs and terminates on its own. gRPC is the natural next step from
// plain protobuf serialization: the .proto file defines the API contract and
// protoc generates both the messages and the client/server stubs.
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/adnvilla/go-examples/examples/grpc/greeterpb"
)

// greeterServer implements the Greeter service from greeter.proto. Embedding
// UnimplementedGreeterServer keeps the implementation forward-compatible:
// RPCs added to the proto later fail with codes.Unimplemented instead of
// breaking the build.
type greeterServer struct {
	greeterpb.UnimplementedGreeterServer
}

// Greet is the unary RPC: validate, then reply once. Errors cross the wire as
// gRPC status codes, so clients can branch on codes.InvalidArgument instead of
// parsing error strings.
func (greeterServer) Greet(_ context.Context, req *greeterpb.GreetRequest) (*greeterpb.GreetReply, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	return &greeterpb.GreetReply{Message: fmt.Sprintf("Hello, %s!", req.GetName())}, nil
}

// Countdown is the server-streaming RPC: one request, a Send per value, and
// returning nil closes the stream — the client sees io.EOF from Recv.
func (greeterServer) Countdown(req *greeterpb.CountdownRequest, stream grpc.ServerStreamingServer[greeterpb.CountdownReply]) error {
	for v := req.GetFrom(); v >= 0; v-- {
		if err := stream.Send(&greeterpb.CountdownReply{Value: v}); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Server side: a random loopback port keeps the demo conflict-free.
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listening: %w", err)
	}

	server := grpc.NewServer()
	greeterpb.RegisterGreeterServer(server, greeterServer{})

	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(lis) }()
	fmt.Println("server: Greeter service up on a loopback port")

	// Client side: NewClient dials lazily; insecure credentials are fine for
	// localhost demos, real deployments use TLS.
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	defer conn.Close() //nolint:errcheck
	client := greeterpb.NewGreeterClient(conn)

	// Unary happy path.
	reply, err := client.Greet(ctx, &greeterpb.GreetRequest{Name: "Ada"})
	if err != nil {
		return fmt.Errorf("Greet: %w", err)
	}
	fmt.Printf("client: Greet(%q) -> %q\n", "Ada", reply.GetMessage())

	// Unary error path: the server's status code survives the wire intact.
	_, err = client.Greet(ctx, &greeterpb.GreetRequest{})
	if st, ok := status.FromError(err); ok && st.Code() == codes.InvalidArgument {
		fmt.Printf("client: Greet(%q) -> code=%s desc=%q\n", "", st.Code(), st.Message())
	} else {
		return fmt.Errorf("Greet with empty name: expected InvalidArgument, got %w", err)
	}

	// Server-streaming: Recv until io.EOF (errors.Is doesn't apply here —
	// gRPC returns io.EOF exactly, and status errors for real failures).
	stream, err := client.Countdown(ctx, &greeterpb.CountdownRequest{From: 3})
	if err != nil {
		return fmt.Errorf("Countdown: %w", err)
	}
	var values []string
	for {
		msg, err := stream.Recv()
		if err != nil {
			break // io.EOF on clean close; a real client would distinguish
		}
		values = append(values, fmt.Sprintf("%d", msg.GetValue()))
	}
	fmt.Printf("client: Countdown(3) -> %s\n", strings.Join(values, " "))

	// GracefulStop waits for in-flight RPCs, then Serve returns nil.
	server.GracefulStop()
	if err := <-serveErr; err != nil {
		return fmt.Errorf("serving: %w", err)
	}
	fmt.Println("server: stopped cleanly")
	return nil
}
