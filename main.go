package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

type LogRecord struct {
	Level     string `json:"level"`
	Job       string `json:"job"`
	Log       string `json:"log"`
	Timestamp string `json:"_timestamp"`
}

var (
	totalBytesSent int64
	batchSize      = 100
	logEndpoint    = ""
	authHeader     = ""
	logRate        = 1 // Logs per second, default
	jobTypes       = []string{
		"user-service",
		"payment-processor",
		"order-management",
		"inventory-service",
		"notification-service",
		"authentication-service",
		"search-service",
		"recommendation-engine",
		"email-service",
		"analytics-processor",
	}
	dbTypes = []string{
		"postgres",
		"mysql",
		"mongodb",
		"redis",
		"elasticsearch",
		"cassandra",
	}
)

func init() {
	if envBatchSize := os.Getenv("BATCH_SIZE"); envBatchSize != "" {
		if size, err := strconv.Atoi(envBatchSize); err == nil {
			batchSize = size
		}
	}
	if envLogRate := os.Getenv("LOG_RATE"); envLogRate != "" {
		if rate, err := strconv.Atoi(envLogRate); err == nil {
			logRate = rate
		}
	}
	if envEndpoint := os.Getenv("LOG_ENDPOINT"); envEndpoint != "" {
		logEndpoint = envEndpoint
	}
	if envAuthHeader := os.Getenv("AUTH_HEADER"); envAuthHeader != "" {
		authHeader = envAuthHeader
	}
}

func getRandomLogLevel() string {
	weights := map[string]int{
		"debug": 15,
		"info":  60,
		"warn":  20,
		"error": 5,
	}
	total := 0
	for _, weight := range weights {
		total += weight
	}

	r := gofakeit.IntRange(1, total)
	current := 0

	for level, weight := range weights {
		current += weight
		if r <= current {
			return level
		}
	}

	return "info"
}

func generateRandomEvent() string {
	events := []string{
		"Processing request from %s",
		"Handled %s request in %dms",
		"Connected to %s",
		"Cache hit for key: %s",
		"Updated user profile for %s",
		"Received webhook from %s",
		"API rate limit: %d requests remaining",
		"Successfully processed transaction %s",
		"Queue size reached %d messages",
		"Memory usage at %d%%",
	}
	eventTemplate := events[gofakeit.IntRange(0, len(events)-1)]

	switch eventTemplate {
	case "Processing request from %s":
		return fmt.Sprintf(eventTemplate, gofakeit.Email())
	case "Handled %s request in %dms":
		return fmt.Sprintf(eventTemplate, gofakeit.HTTPMethod(), gofakeit.IntRange(10, 500))
	case "Connected to %s":
		return fmt.Sprintf(eventTemplate, dbTypes[gofakeit.IntRange(0, len(dbTypes)-1)])
	case "Cache hit for key: %s":
		return fmt.Sprintf(eventTemplate, gofakeit.UUID())
	case "Updated user profile for %s":
		return fmt.Sprintf(eventTemplate, gofakeit.Username())
	case "Received webhook from %s":
		return fmt.Sprintf(eventTemplate, gofakeit.Company())
	case "API rate limit: %d requests remaining":
		return fmt.Sprintf(eventTemplate, gofakeit.IntRange(1, 1000))
	case "Successfully processed transaction %s":
		return fmt.Sprintf(eventTemplate, gofakeit.UUID())
	case "Queue size reached %d messages":
		return fmt.Sprintf(eventTemplate, gofakeit.IntRange(100, 10000))
	case "Memory usage at %d%%":
		return fmt.Sprintf(eventTemplate, gofakeit.IntRange(20, 95))
	default:
		return "Default log message"
	}
}

func generateLogData(wg *sync.WaitGroup, client *http.Client, done chan bool) {
	defer wg.Done()
	delay := time.Second / time.Duration(logRate)
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			batch := make([]LogRecord, batchSize)
			now := time.Now()

			for i := 0; i < batchSize; i++ {
				batch[i] = LogRecord{
					Level:     getRandomLogLevel(),
					Job:       jobTypes[gofakeit.IntRange(0, len(jobTypes)-1)],
					Log:       generateRandomEvent(),
					Timestamp: now.Format(time.RFC3339),
				}
			}
			sendLogBatch(client, batch)
		}
	}
}

func sendLogBatch(client *http.Client, logBatch []LogRecord) {
	batchData, err := json.Marshal(logBatch)
	if err != nil {
		log.Printf("failed to marshal log batch: %v", err)
		return
	}

	req, err := http.NewRequest("POST", logEndpoint, bytes.NewBuffer(batchData))
	if err != nil {
		log.Printf("failed to create HTTP request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to send log batch: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("unexpected status code: %d", resp.StatusCode)
		return
	}

	atomic.AddInt64(&totalBytesSent, int64(len(batchData)))
}

func displayMBps() {
	startTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		duration := time.Since(startTime).Seconds()
		bytesSent := atomic.LoadInt64(&totalBytesSent)
		mbSent := float64(bytesSent) / (1024 * 1024)
		mbps := mbSent / duration
		fmt.Printf("Data transfer rate: %.2f MB/s\n", mbps)
	}
}

func main() {
	gofakeit.Seed(0)

	var wg sync.WaitGroup
	done := make(chan bool)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	go displayMBps()

	for i := 0; i < logRate; i++ {
		wg.Add(1)
		go generateLogData(&wg, client, done)
	}

	// Keep the program running indefinitely
	select {}
}
