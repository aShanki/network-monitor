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

	log.Println("Starting network monitor...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Loaded Configuration: Interface='%s', Threshold=%.2f Mbps, Interval=%ds, Webhook Set: %t, TopN: %d",
		cfg.InterfaceName, cfg.ThresholdMbps, cfg.IntervalSeconds, cfg.WebhookURL != "", cfg.TopN)

	m, err := monitor.NewMonitor(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize monitor: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go m.Run()

	log.Println("Monitor started. Press Ctrl+C to stop.")

	<-sigChan

	log.Println("Shutdown signal received, stopping monitor...")
	m.Close()

	log.Println("Monitor stopped gracefully.")
}
