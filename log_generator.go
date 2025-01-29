package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "math/rand"
    "net/http"
    "os"
    "strconv"
    "sync"
    "sync/atomic"
    "time"

    "github.com/brianvoe/gofakeit/v6"
)

// LogRecord represents a single log entry
type LogRecord struct {
    Level     string `json:"level"`
    Job       string `json:"job"`
    Log       string `json:"log"`
    Timestamp string `json:"_timestamp"`
}

// Global variables
var (
    totalBytesSent int64
    jobTypes       = []string{
        "user-service", "payment-processor", "order-management",
        "inventory-service", "notification-service", "authentication-service",
        "search-service", "recommendation-engine", "email-service", "analytics-processor",
    }
    dbTypes = []string{"postgres", "mysql", "mongodb", "redis", "elasticsearch", "cassandra"}
    config  struct {
        LogEndpoint string
        AuthHeader  string
        LogRate     int
        BatchSize   int
    }
)

func init() {
    config.LogEndpoint = os.Getenv("LOG_ENDPOINT")
    if config.LogEndpoint == "" {
        log.Fatal("LOG_ENDPOINT environment variable is required")
    }
    config.AuthHeader = os.Getenv("AUTH_HEADER")
    config.LogRate = getEnvInt("LOG_RATE", 1)
    config.BatchSize = getEnvInt("BATCH_SIZE", 100)

    // Initialize random seed
    rand.Seed(time.Now().UnixNano())
}

// getEnvInt retrieves an integer from environment variables with a default value
func getEnvInt(key string, defaultValue int) int {
    if val, err := strconv.Atoi(os.Getenv(key)); err == nil {
        return val
    }
    return defaultValue
}

// getRandomLogLevel returns a random log level based on weighted distribution
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

    r := rand.Intn(total)
    current := 0
    for level, weight := range weights {
        current += weight
        if r < current {
            return level
        }
    }
    return "info"
}

// generateRandomEvent creates a random log message
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
    eventTemplate := events[rand.Intn(len(events))]

    switch eventTemplate {
    case "Processing request from %s":
        return fmt.Sprintf(eventTemplate, gofakeit.Email())
    case "Handled %s request in %dms":
        return fmt.Sprintf(eventTemplate, gofakeit.HTTPMethod(), rand.Intn(490)+10)
    case "Connected to %s":
        return fmt.Sprintf(eventTemplate, dbTypes[rand.Intn(len(dbTypes))])
    case "Cache hit for key: %s":
        return fmt.Sprintf(eventTemplate, gofakeit.UUID())
    case "Updated user profile for %s":
        return fmt.Sprintf(eventTemplate, gofakeit.Email())
    case "Received webhook from %s":
        return fmt.Sprintf(eventTemplate, gofakeit.URL())
    case "API rate limit: %d requests remaining":
        return fmt.Sprintf(eventTemplate, rand.Intn(1000))
    case "Successfully processed transaction %s":
        return fmt.Sprintf(eventTemplate, gofakeit.UUID())
    case "Queue size reached %d messages":
        return fmt.Sprintf(eventTemplate, rand.Intn(10000))
    case "Memory usage at %d%%":
        return fmt.Sprintf(eventTemplate, rand.Intn(100))
    default:
        return "Default log message"
    }
}

// generateLogData continuously generates and sends log data
func generateLogData(wg *sync.WaitGroup, client *http.Client, done chan bool) {
    defer wg.Done()
    ticker := time.NewTicker(time.Second / time.Duration(config.LogRate))
    defer ticker.Stop()

    for {
        select {
        case <-done:
            return
        case <-ticker.C:
            batch := make([]LogRecord, config.BatchSize)
            now := time.Now()

            for i := 0; i < config.BatchSize; i++ {
                batch[i] = LogRecord{
                    Level:     getRandomLogLevel(),
                    Job:       jobTypes[rand.Intn(len(jobTypes))],
                    Log:       generateRandomEvent(),
                    Timestamp: now.Format(time.RFC3339),
                }
            }
            
            if err := sendLogBatch(client, batch); err != nil {
                log.Printf("Failed to send log batch: %v", err)
            }
        }
    }
}

// sendLogBatch sends a batch of logs to the configured endpoint
func sendLogBatch(client *http.Client, logBatch []LogRecord) error {
    batchData, err := json.Marshal(logBatch)
    if err != nil {
        return fmt.Errorf("failed to marshal log batch: %w", err)
    }

    req, err := http.NewRequest("POST", config.LogEndpoint, bytes.NewBuffer(batchData))
    if err != nil {
        return fmt.Errorf("failed to create HTTP request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")
    if config.AuthHeader != "" {
        req.Header.Set("Authorization", config.AuthHeader)
    }

    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send log batch: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("server returned error status: %d", resp.StatusCode)
    }

    atomic.AddInt64(&totalBytesSent, int64(len(batchData)))
    return nil
}