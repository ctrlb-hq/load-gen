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

var (
	endpoint   = getEnvOrDefault("TRACE_ENDPOINT", "http://localhost:4318/traces")
	authHeader = getEnvOrDefault("AUTH_HEADER", "")
	client     = &http.Client{Timeout: 10 * time.Second}
)

var serviceNames = []string{"user-service", "order-service", "payment-service", "inventory-service"}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
	payload, err := json.Marshal(trace)
	if err != nil {
		return fmt.Errorf("error marshaling trace: %v", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending trace: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func generateTrace() error {
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

		// Simulate processing time
		time.Sleep(time.Millisecond * time.Duration(100+mathrand.Intn(200)))
		childSpan.EndTime = time.Now().UnixNano()
		trace.Spans = append(trace.Spans, childSpan)
	}

	rootSpan.EndTime = time.Now().UnixNano()
	trace.Spans = append(trace.Spans, rootSpan)

	return sendTrace(trace)
}

func startTraceGeneration() error {
	ctx := context.Background()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := generateTrace(); err != nil {
				log.Printf("Error generating trace: %v", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
