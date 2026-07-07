// Demonstrates distributed trace propagation: two HTTP services in one
// process, where the caller Injects the W3C traceparent header and the
// callee Extracts it, so spans created in different services join one trace.
// The [otel] example covers spans within a process; this one shows the
// mechanism that makes tracing *distributed* — and prints the actual header
// that crosses the wire.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// In-memory exporter + synchronous processor: the demo can inspect every
	// finished span and print a compact summary instead of raw JSON.
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { err = errors.Join(err, provider.Shutdown(context.WithoutCancel(ctx))) }()
	otel.SetTracerProvider(provider)

	// The propagator is the codec for trace context on the wire. W3C
	// TraceContext is the standard: one `traceparent` header.
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Service B: "inventory" — extracts the incoming trace context.
	inventory, err := startServer(ctx, http.HandlerFunc(handleInventory))
	if err != nil {
		return err
	}
	defer inventory.Close()

	// Service A: "checkout" — starts the trace and calls inventory.
	checkout, err := startServer(ctx, checkoutHandler(inventory.URL))
	if err != nil {
		return err
	}
	defer checkout.Close()

	fmt.Println("--- two services, one trace ---")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkout.URL, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling checkout: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	fmt.Printf("client got: %s", body)

	return printSpanSummary(exporter)
}

// checkoutHandler is service A: it starts a server span (the trace root,
// since the demo's client sends no traceparent), then calls service B with
// the trace context Injected into the outgoing headers.
func checkoutHandler(inventoryURL string) http.HandlerFunc {
	tracer := otel.Tracer("checkout-service")
	return func(w http.ResponseWriter, r *http.Request) {
		// No incoming traceparent -> this span starts a fresh trace.
		ctx, span := tracer.Start(r.Context(), "checkout", trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, inventoryURL, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Inject serializes the CURRENT span context into the headers —
		// this line is the entire "distributed" in distributed tracing.
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
		fmt.Printf("checkout: calling inventory with traceparent=%s\n", req.Header.Get("traceparent"))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close() //nolint:errcheck
		if _, err := io.Copy(w, resp.Body); err != nil {
			return // response already partially written; nothing to add
		}
	}
}

// handleInventory is service B: it Extracts the trace context from the
// incoming headers, so its span joins the caller's trace instead of
// starting a new one.
func handleInventory(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("inventory-service")

	// Extract deserializes traceparent into a context carrying the remote
	// span context; without this line the services produce two unrelated
	// traces and the waterfall view falls apart.
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	_, span := tracer.Start(ctx, "reserve-inventory", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	remote := trace.SpanContextFromContext(ctx)
	fmt.Printf("inventory: extracted remote context (trace %s...) — joining the caller's trace\n",
		remote.TraceID().String()[:8])

	time.Sleep(5 * time.Millisecond) // simulated work
	_, _ = fmt.Fprintln(w, "reserved")
}

// printSpanSummary shows the payoff: every exported span, across both
// services, carries the same trace ID and chains parent to child.
func printSpanSummary(exporter *tracetest.InMemoryExporter) error {
	fmt.Println("\n--- exported spans ---")
	spans := exporter.GetSpans()
	if len(spans) != 2 {
		return fmt.Errorf("exported %d spans, want 2", len(spans))
	}

	traceIDs := map[string]bool{}
	for _, s := range spans {
		traceIDs[s.SpanContext.TraceID().String()] = true
		parent := "none (root)"
		if s.Parent.IsValid() {
			parent = "span " + s.Parent.SpanID().String()[:8] + "... (remote=" +
				fmt.Sprint(s.Parent.IsRemote()) + ")"
		}
		fmt.Printf("%-18s trace=%s... parent=%s\n",
			s.Name, s.SpanContext.TraceID().String()[:8], parent)
	}
	fmt.Printf("all spans share one trace id: %t\n", len(traceIDs) == 1)
	if len(traceIDs) != 1 {
		return errors.New("propagation failed: spans landed in different traces")
	}
	return nil
}

// startServer runs an http.Server for the given handler on a loopback port.
type server struct {
	URL   string
	close func()
}

func (s *server) Close() { s.close() }

func startServer(ctx context.Context, handler http.Handler) (*server, error) {
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listening: %w", err)
	}
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	go srv.Serve(lis) //nolint:errcheck // closed via srv.Close in teardown
	return &server{
		URL:   "http://" + lis.Addr().String(),
		close: func() { _ = srv.Close() },
	}, nil
}
