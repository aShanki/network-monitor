package main

import (
	"log"
	"network-monitor/internal/config"
	"network-monitor/internal/monitor"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Define command-line flags (these will be overridden by config file/env vars if present)
	// flag.String("i", "", "Network interface to monitor (e.g., eth0). If empty, chooses the first available non-loopback interface.")
	// flag.Float64("t", 1024*1024, "Speed threshold in Bytes per second (B/s).")         // Default: 1 MB/s
	// flag.Duration("interval", 5*time.Second, "Monitoring interval (e.g., 1s, 5m, 1h).") // Default: 5 seconds
	// flag.Parse() // Parsing handled by config loader

	log.Println("Starting network monitor...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Log final configuration (replace old logging)
	// log.Printf("Configuration: Interface='%s', Threshold=%.2f B/s, Interval=%s", *interfaceName, *threshold, (*interval).String())
	log.Printf("Loaded Configuration: Interface='%s', Threshold=%.2f Mbps, Interval=%ds, Webhook Set: %t, TopN: %d",
		cfg.InterfaceName, cfg.ThresholdMbps, cfg.IntervalSeconds, cfg.WebhookURL != "", cfg.TopN)

	// Create the monitor using the loaded config
	m, err := monitor.NewMonitor(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize monitor: %v", err)
	}

	// Set up channel to listen for OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run the monitor in a separate goroutine
	go m.Run()

	log.Println("Monitor started. Press Ctrl+C to stop.")

	// Wait for a shutdown signal
	<-sigChan

	// Initiate graceful shutdown
	log.Println("Shutdown signal received, stopping monitor...")
	m.Close() // Close the monitor gracefully (stops aggregator, etc.)

	log.Println("Monitor stopped gracefully.")
}
