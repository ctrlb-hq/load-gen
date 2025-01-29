package main

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	mathrand "math/rand"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Endpoint string            `json:"endpoint"`
	Headers  map[string]string `json:"headers"`
}

var (
	defaultConfig = Config{
		Endpoint: "http://localhost:4318/traces",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"stream-name":  "default",
		},
	}
	tracesConfig = loadConfig()
	client       = &http.Client{Timeout: 10 * time.Second}
)

var serviceNames = []string{"user-service", "order-service", "payment-service", "inventory-service"}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func loadConfig() Config {
	cfg := defaultConfig
	log.Println("Loading trace configuration...")

	if endpoint := os.Getenv("TRACES_ENDPOINT"); endpoint != "" {
		log.Printf("Using custom endpoint: %s", endpoint)
		cfg.Endpoint = endpoint
	}

	if auth := os.Getenv("AUTH_HEADER"); auth != "" {
		log.Println("Authorization header found")
		cfg.Headers["Authorization"] = auth
	}

	if stream := os.Getenv("TRACES_STREAM"); stream != "" {
		log.Printf("Using stream: %s", stream)
		cfg.Headers["stream-name"] = stream
	}

	return cfg
}

func generateRandomID() string {
	bytes := make([]byte, 16)
	_, err := cryptorand.Read(bytes)
	if err != nil {
		log.Fatalf("error reading random bytes: %v", err)
	}
	return fmt.Sprintf("%x", bytes)
}

// Trace structures
type Span struct {
	TraceID     string            `json:"traceId"`
	SpanID      string            `json:"spanId"`
	ParentID    string            `json:"parentId,omitempty"`
	Name        string            `json:"name"`
	StartTime   int64             `json:"startTime"`
	EndTime     int64             `json:"endTime"`
	ServiceName string            `json:"serviceName"`
	Attributes  map[string]string `json:"attributes"`
}

type Trace struct {
	Spans []Span `json:"spans"`
}

func sendTrace(trace *Trace) error {
	log.Printf("Sending trace with %d spans...", len(trace.Spans))
	payload, err := json.Marshal(trace)
	if err != nil {
	}

	req, err := http.NewRequest("POST", tracesConfig.Endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	fmt.Printf("Auth Header: %v\n", tracesConfig.Headers["Authorization"])
	fmt.Println("Endpoint: ", tracesConfig.Endpoint)
	// Set all configured headers
	for key, value := range tracesConfig.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending trace: %v", err)
		return fmt.Errorf("error sending trace: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status code: %d", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	log.Printf("Successfully sent trace with %d spans", len(trace.Spans))
	return nil
}

func generateTrace(ctx context.Context) error {
	traceID := generateRandomID()
	trace := &Trace{Spans: make([]Span, 0)}
	now := time.Now()

	// Root span
	rootSpan := Span{
		TraceID:     traceID,
		SpanID:      generateRandomID(),
		Name:        "API Request",
		StartTime:   now.UnixNano(),
		ServiceName: "trace-generator",
		Attributes:  map[string]string{"span.kind": "server"},
	}

	// Process services
	for _, service := range serviceNames {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			childSpan := Span{
				TraceID:     traceID,
				SpanID:      generateRandomID(),
				ParentID:    rootSpan.SpanID,
				Name:        service,
				StartTime:   now.Add(time.Duration(mathrand.Intn(100)) * time.Millisecond).UnixNano(),
				ServiceName: service,
				Attributes: map[string]string{
					"span.kind":    "client",
					"operation":    "process_request",
					"service.name": service,
				},
			}

			// Replace time.Sleep with context-aware sleep
			timer := time.NewTimer(time.Millisecond * time.Duration(100+mathrand.Intn(200)))
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}

			childSpan.EndTime = time.Now().UnixNano()
			trace.Spans = append(trace.Spans, childSpan)
		}
	}

	rootSpan.EndTime = time.Now().UnixNano()
	trace.Spans = append(trace.Spans, rootSpan)

	return sendTrace(trace)
}

func startTraceGeneration(ctx context.Context) error {
	log.Println("Starting trace generation...")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	traceCount := 0
	for {
		select {
		case <-ticker.C:
			traceCount++
			log.Printf("Generating trace #%d", traceCount)
			if err := generateTrace(ctx); err != nil {
				if err == context.Canceled {
					log.Println("Trace generation canceled")
					return err
				}
				log.Printf("Error generating trace #%d: %v", traceCount, err)
			}
		case <-ctx.Done():
			log.Println("Stopping trace generation...")
			return ctx.Err()
		}
	}
}
