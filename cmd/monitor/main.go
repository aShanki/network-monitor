package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"network-monitor/internal/analysis"
	"network-monitor/internal/capture"
)

// Temporary Config struct until Task 3.2 is done
type TempConfig struct {
	InterfaceName   string
	IntervalSeconds int
	// Add other fields like Threshold, WebhookURL later
}

func main() {
	log.Println("Starting network monitor...")

	// TODO: Replace with proper config loading (Task 3.2)
	cfg := &TempConfig{
		InterfaceName:   "", // Auto-detect
		IntervalSeconds: 5,  // Example: 5-second interval
	}

	// Use configured interface name
	packetSource, handle, err := capture.StartCapture(cfg.InterfaceName)
	if err != nil {
		log.Fatalf("Failed to start capture: %v", err)
	}
	defer handle.Close()

	log.Println("Packet capture started successfully.")

	// Setup WaitGroup for graceful shutdown
	var wg sync.WaitGroup

	// Create Aggregator
	// Convert TempConfig to the expected config.Config for NewAggregator
	// This requires defining the config.Config struct or adjusting NewAggregator signature temporarily
	// Let's assume config.Config has at least IntervalSeconds for now
	// We'll need to create internal/config/config.go eventually
	// For now, let's simulate it inline or assume NewAggregator is flexible enough
	// --- SIMULATION START (Replace with actual config pkg later) ---
	// type SimConfig struct { // Simulate config.Config
	// 	IntervalSeconds int
	// }
	// simCfg := &SimConfig{IntervalSeconds: cfg.IntervalSeconds} // Remove unused variable
	// --- SIMULATION END ---

	// Pass packetSource and simulated config to NewAggregator
	// Note: NewAggregator expects *config.Config, we'll need to adjust this or create the actual config package
	// For demonstration, let's imagine NewAggregator could take SimConfig (or we adjust it)
	// Assuming NewAggregator needs adjustment or config package is created:
	// Assuming analysis.NewAggregator is updated to take an interface or specific fields.
	// Let's proceed assuming we'll create config.go next or NewAggregator is adjusted.
	// For now, this might cause a compile error until config is sorted.

	// We need a logger
	logger := log.New(os.Stdout, "ANALYSIS: ", log.LstdFlags)

	// Start the aggregator - this now handles packet reading internally
	aggregator, resultsChan := analysis.NewAggregator(
		&analysis.ConfigForAggregator{IntervalSeconds: cfg.IntervalSeconds}, // Pass necessary fields directly
		packetSource,
		logger,
	)

	log.Println("Traffic aggregator started.")

	// Goroutine to handle analysis results
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Result processing goroutine started.")
		for intervalData := range resultsChan {
			// TODO: Implement Task 3.5 (Threshold check, Top Talkers)
			log.Printf("Received interval data with %d IP entries\n", len(intervalData))
			// Example: Print the data
			// for ip, data := range intervalData {
			// 	 log.Printf("  IP: %s, Bytes: %d\n", ip, data.Bytes)
			// }
		}
		log.Println("Result processing goroutine finished.")
	}()

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Running. Press Ctrl+C to stop.")
	<-sigChan // Block until a signal is received

	log.Println("Shutdown signal received, stopping components...")

	// Signal the aggregator to stop
	// This will close its internal stopChan, stopping its goroutines
	// and eventually closing the resultsChan
	aggregator.Stop()

	// The capture handle is closed by defer handle.Close()

	// Wait for the results processing goroutine to finish
	log.Println("Waiting for goroutines to finish...")
	wg.Wait()

	log.Println("Network monitor stopped gracefully.")
}
