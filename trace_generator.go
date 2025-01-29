package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer
var serviceNames = []string{"user-service", "order-service", "payment-service", "inventory-service"}

// Reads environment variables
var (
	otelEndpoint = getEnvOrDefault("OTEL_TRACE_ENDPOINT", "localhost:4318")
	authHeader   = getEnvOrDefault("AUTH_HEADER", "")
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Initializes the OpenTelemetry tracer provider
func initTracer() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	// Parse endpoint to separate host and path
	endpoint := strings.TrimPrefix(otelEndpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	headers := map[string]string{
		"Authorization": authHeader,
		"stream-name":   "default",
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithHeaders(headers),
		otlptracehttp.WithInsecure(), // Remove this if using HTTPS
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %v", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("trace-generator"),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("stream-name", "default"),
		),
		resource.WithSchemaURL(semconv.SchemaURL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %v", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("trace-generator")

	return tp, nil
}

// Simulates a trace flow across multiple services
func generateTrace(ctx context.Context) error {
	ctx, rootSpan := tracer.Start(ctx, "API Request")
	defer rootSpan.End()

	rootSpan.SetAttributes(
		attribute.String("trace_id", rootSpan.SpanContext().TraceID().String()),
		attribute.String("span.kind", "server"),
	)

	// Simulating interlinked spans across services
	for _, service := range serviceNames {
		if err := processService(ctx, service); err != nil {
			return fmt.Errorf("error processing service %s: %v", service, err)
		}
	}

	return nil
}

func processService(ctx context.Context, serviceName string) error {
	ctx, span := tracer.Start(ctx, serviceName)
	defer span.End()

	span.SetAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("operation", "process_request"),
		attribute.String("span.kind", "client"),
	)

	// Simulate processing time with some randomness
	time.Sleep(time.Millisecond * time.Duration(100+rand.Intn(200)))

	return nil
}

// Starts the infinite trace generation
func startTraceGeneration() error {
	tp, err := initTracer()
	if err != nil {
		return fmt.Errorf("failed to initialize tracer: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	ctx := context.Background()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := generateTrace(ctx); err != nil {
				log.Printf("Error generating trace: %v", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
