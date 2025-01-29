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

// Move shared configuration to a separate package or config struct
var config struct {
    LogEndpoint string
    AuthHeader  string
    LogRate     int
    BatchSize   int
}

func init() {
    config.LogEndpoint = os.Getenv("LOG_ENDPOINT")
    if config.LogEndpoint == "" {
        log.Fatal("LOG_ENDPOINT environment variable is required")
    }
    config.AuthHeader = os.Getenv("AUTH_HEADER")
    config.LogRate = getEnvInt("LOG_RATE", 1)
    config.BatchSize = getEnvInt("BATCH_SIZE", 100)
}

// Rest of your existing LogRecord struct and variables...

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
                    Job:       jobTypes[gofakeit.IntRange(0, len(jobTypes)-1)],
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