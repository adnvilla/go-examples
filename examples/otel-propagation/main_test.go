package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// setupTracing installs an in-memory exporter and the W3C propagator for the
// duration of one test. Global state forces these tests to run sequentially.
func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() { _ = provider.Shutdown(t.Context()) })
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return exporter
}

func TestSpansAcrossServicesShareOneTrace(t *testing.T) {
	exporter := setupTracing(t)

	inventory := httptest.NewServer(http.HandlerFunc(handleInventory))
	defer inventory.Close()
	checkout := httptest.NewServer(checkoutHandler(inventory.URL))
	defer checkout.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, checkout.URL, nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("calling checkout: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("reading response: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("exported %d spans, want 2", len(spans))
	}
	if a, b := spans[0].SpanContext.TraceID(), spans[1].SpanContext.TraceID(); a != b {
		t.Fatalf("trace ids differ: %s vs %s — propagation broken", a, b)
	}
}

func TestInventorySpanHasRemoteParent(t *testing.T) {
	exporter := setupTracing(t)

	inventory := httptest.NewServer(http.HandlerFunc(handleInventory))
	defer inventory.Close()
	checkout := httptest.NewServer(checkoutHandler(inventory.URL))
	defer checkout.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, checkout.URL, nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("calling checkout: %v", err)
	}
	_ = resp.Body.Close()

	for _, s := range exporter.GetSpans() {
		switch s.Name {
		case "reserve-inventory":
			if !s.Parent.IsValid() || !s.Parent.IsRemote() {
				t.Errorf("inventory span parent: valid=%t remote=%t, want a remote parent",
					s.Parent.IsValid(), s.Parent.IsRemote())
			}
		case "checkout":
			if s.Parent.IsValid() {
				t.Errorf("checkout span should be the trace root, has parent %s", s.Parent.SpanID())
			}
		}
	}
}

func TestTraceparentHeaderCrossesTheWire(t *testing.T) {
	setupTracing(t)

	var gotHeader string
	inventory := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("traceparent")
		handleInventory(w, r)
	}))
	defer inventory.Close()
	checkout := httptest.NewServer(checkoutHandler(inventory.URL))
	defer checkout.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, checkout.URL, nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("calling checkout: %v", err)
	}
	_ = resp.Body.Close()

	// W3C format: version-traceid-spanid-flags = 2-32-16-2 hex chars.
	if len(gotHeader) != 55 {
		t.Fatalf("traceparent = %q (len %d), want the 55-char W3C format", gotHeader, len(gotHeader))
	}
}
