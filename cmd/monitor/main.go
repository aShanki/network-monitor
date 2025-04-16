package main

import (
	"flag"
	"log"
	"network-monitor/internal/monitor" // Adjust import path if necessary
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Define command-line flags
	interfaceName := flag.String("i", "", "Network interface to monitor (e.g., eth0). If empty, chooses the first available non-loopback interface.")
	threshold := flag.Float64("t", 1024*1024, "Speed threshold in Bytes per second (B/s).")         // Default: 1 MB/s
	interval := flag.Duration("interval", 5*time.Second, "Monitoring interval (e.g., 1s, 5m, 1h).") // Default: 5 seconds

	flag.Parse()

	log.Println("Starting network monitor...")
	log.Printf("Configuration: Interface='%s', Threshold=%.2f B/s, Interval=%s", *interfaceName, *threshold, (*interval).String())

	// Create the monitor
	m, err := monitor.NewMonitor(*interfaceName, *threshold, *interval)
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
	m.Close() // Close the pcap handle
	log.Println("Monitor stopped gracefully.")
}
