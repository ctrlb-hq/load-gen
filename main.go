package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"
)

func main() {
    // Setup graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    var wg sync.WaitGroup
    done := make(chan bool)
    client := &http.Client{Timeout: 10 * time.Second}

    // Start log generation
    wg.Add(1)
    go generateLogData(&wg, client, done)

    // Start trace generation
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := startTraceGeneration(); err != nil {
            log.Printf("Trace generation failed: %v", err)
            cancel()
        }
    }()

    // Wait for shutdown signal
    select {
    case sig := <-sigChan:
        log.Printf("Received signal: %v", sig)
        cancel()
    case <-ctx.Done():
        log.Println("Context cancelled")
    }

    // Initiate shutdown
    close(done)
    log.Println("Waiting for goroutines to finish...")
    wg.Wait()
    log.Println("Shutdown complete")
}