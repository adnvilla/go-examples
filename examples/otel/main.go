// Demonstrates distributed tracing with OpenTelemetry: a TracerProvider wired
// to an exporter, nested spans propagated through context, attributes, events,
// and error recording. Runs self-contained by default (spans print to stdout);
// point OTEL_EXPORTER_OTLP_ENDPOINT at Jaeger to see the same trace in a UI.
// Tracing completes the observability triad next to slog (logs) and metric.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var errCardDeclined = errors.New("card declined")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// newExporter picks the span destination: OTLP/HTTP when the standard
// OTEL_EXPORTER_OTLP_ENDPOINT variable is set (e.g. Jaeger on :4318),
// stdout otherwise — so the example runs with zero infrastructure.
func newExporter(ctx context.Context) (sdktrace.SpanExporter, string, error) {
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		exporter, err := otlptracehttp.New(ctx)
		return exporter, "OTLP -> " + endpoint, err
	}
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	return exporter, "stdout (set OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 for Jaeger)", err
}

func run() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exporter, destination, err := newExporter(ctx)
	if err != nil {
		return fmt.Errorf("creating exporter: %w", err)
	}

	// The TracerProvider is the SDK's assembly point: exporter (where spans
	// go), batcher (buffers and ships them off the hot path), and resource
	// (identity attributes stamped on every span, service.name above all).
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewSchemaless(
			semconv.ServiceName("checkout-demo"),
		)),
	)
	// Shutdown flushes the batcher; skip it and the process exits with spans
	// still sitting in the buffer.
	defer func() { err = errors.Join(err, provider.Shutdown(context.WithoutCancel(ctx))) }()
	otel.SetTracerProvider(provider)

	tracer := otel.Tracer("github.com/adnvilla/go-examples/examples/otel")
	fmt.Println("exporting spans via:", destination)

	// One successful checkout and one with a declined card — the failure is
	// recorded on the trace (status + error event), not just returned.
	if err := checkout(ctx, tracer, "order-1001", false); err != nil {
		return err
	}
	fmt.Println("checkout order-1001: completed")

	if err := checkout(ctx, tracer, "order-1002", true); err != nil {
		fmt.Printf("checkout order-1002: failed: %v (failure recorded on the trace)\n", err)
	}
	return nil
}

// checkout is the root span of each trace. The context returned by Start
// carries the span, so child operations called with it become child spans —
// that context flow IS the trace structure.
func checkout(ctx context.Context, tracer trace.Tracer, orderID string, declineCard bool) error {
	ctx, span := tracer.Start(ctx, "checkout",
		trace.WithAttributes(attribute.String("order.id", orderID)))
	defer span.End()

	span.AddEvent("order received")

	if err := reserveInventory(ctx, tracer, orderID); err != nil {
		span.SetStatus(codes.Error, "inventory reservation failed")
		return fmt.Errorf("reserving inventory: %w", err)
	}
	if err := chargeCard(ctx, tracer, declineCard); err != nil {
		span.SetStatus(codes.Error, "payment failed")
		return fmt.Errorf("charging card: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func reserveInventory(ctx context.Context, tracer trace.Tracer, orderID string) error {
	_, span := tracer.Start(ctx, "reserve-inventory",
		trace.WithAttributes(attribute.Int("items.count", 2)))
	defer span.End()

	time.Sleep(5 * time.Millisecond) // simulated warehouse call
	span.AddEvent("reservation confirmed", trace.WithAttributes(
		attribute.String("order.id", orderID)))
	return nil
}

func chargeCard(ctx context.Context, tracer trace.Tracer, decline bool) error {
	_, span := tracer.Start(ctx, "charge-card")
	defer span.End()

	time.Sleep(8 * time.Millisecond) // simulated payment-gateway call
	if decline {
		// RecordError attaches the error as a span event; SetStatus marks the
		// span failed. Backends use the status for error rates and filtering.
		span.RecordError(errCardDeclined)
		span.SetStatus(codes.Error, errCardDeclined.Error())
		return errCardDeclined
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
