package main

import (
	"log"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"network-monitor/internal/analysis"
	"network-monitor/internal/capture"
	// "network-monitor/internal/discord" // Keep commented until Task 3.6
)

// Temporary Config struct until Task 3.2 is done
type TempConfig struct {
	InterfaceName   string
	IntervalSeconds int
	ThresholdMbps   float64
	TopN            int
	// Add WebhookURL later
}

func main() {
	log.Println("Starting network monitor...")

	// TODO: Replace with proper config loading (Task 3.2)
	cfg := &TempConfig{
		InterfaceName:   "",   // Auto-detect
		IntervalSeconds: 5,    // Example: 5-second interval
		ThresholdMbps:   10.0, // Example: 10 Mbps threshold
		TopN:            5,    // Example: Top 5 IPs
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
		// Get interval duration from config for speed calculation
		intervalDuration := time.Duration(cfg.IntervalSeconds) * time.Second

		for intervalData := range resultsChan {
			// --- Task 3.5 Implementation START ---

			// Calculate total bytes for the interval
			totalBytes := int64(0)
			for _, data := range intervalData {
				totalBytes += data.Bytes
			}

			// Calculate overall speed for the interval
			overallSpeedMbps := analysis.CalculateSpeedMbps(totalBytes, intervalDuration)
			log.Printf("Interval finished. Total Bytes: %d, Overall Speed: %.2f Mbps\n", totalBytes, overallSpeedMbps)

			// Check against threshold
			if overallSpeedMbps > cfg.ThresholdMbps {
				log.Printf("ALERT: Overall speed %.2f Mbps exceeded threshold of %.2f Mbps\n", overallSpeedMbps, cfg.ThresholdMbps)

				// Identify Top Talkers
				type ipTraffic struct {
					IP    string
					Bytes int64
				}

				// Convert map to slice for sorting
				trafficSlice := make([]ipTraffic, 0, len(intervalData))
				for ip, data := range intervalData {
					trafficSlice = append(trafficSlice, ipTraffic{IP: ip, Bytes: data.Bytes})
				}

				// Sort by bytes descending
				sort.Slice(trafficSlice, func(i, j int) bool {
					return trafficSlice[i].Bytes > trafficSlice[j].Bytes
				})

				// Get Top N
				numTalkers := cfg.TopN
				if len(trafficSlice) < numTalkers {
					numTalkers = len(trafficSlice)
				}
				topTalkersSlice := trafficSlice[:numTalkers]

				// Prepare data for notification (IP -> Speed Mbps)
				topTalkersResult := make(map[string]float64)
				log.Println("--- Top Talkers ---")
				for _, talker := range topTalkersSlice {
					speedMbps := analysis.CalculateSpeedMbps(talker.Bytes, intervalDuration)
					topTalkersResult[talker.IP] = speedMbps
					log.Printf("  IP: %s, Speed: %.2f Mbps (Bytes: %d)\n", talker.IP, speedMbps, talker.Bytes)
				}
				log.Println("-------------------")

				// TODO: Task 3.6 - Send topTalkersResult to Discord
				// discord.SendNotification(cfg.WebhookURL, topTalkersResult)

			}
			// --- Task 3.5 Implementation END ---
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
