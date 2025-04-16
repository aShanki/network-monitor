package main

import (
	"context"
	"log"
	"network-monitor/internal/capture"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/gopacket"
)

func main() {
	log.Println("Starting network monitor...")

	// Interface name can be overridden by config/flags later.
	// Pass empty string to auto-detect.
	interfaceName := ""

	packetSource, handle, err := capture.StartCapture(interfaceName)
	if err != nil {
		log.Fatalf("Failed to start capture: %v", err)
	}
	defer handle.Close()

	log.Println("Packet capture started successfully. Waiting for packets...")

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancel is called eventually

	// Goroutine to process packets
	go processPackets(ctx, packetSource)

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Running. Press Ctrl+C to stop.")
	<-sigChan // Block until a signal is received

	log.Println("Shutdown signal received, stopping capture...")
	// Cancel the context to signal the packet processing goroutine to stop
	cancel()

	// Allow some time for goroutines to clean up
	// In a real app, might use sync.WaitGroup
	time.Sleep(1 * time.Second)
	log.Println("Network monitor stopped.")
}

// processPackets reads packets from the source and handles them.
// It stops when the context is cancelled.
func processPackets(ctx context.Context, source *gopacket.PacketSource) {
	packetCount := 0
	log.Println("Packet processing goroutine started.")
	for {
		select {
		case <-ctx.Done(): // Check if the context has been cancelled
			log.Printf("Packet processing stopped. Total packets processed: %d\n", packetCount)
			return
		case packet, ok := <-source.Packets():
			if !ok {
				log.Println("Packet source channel closed.")
				return // Exit if the channel is closed
			}
			packetCount++
			// TODO: Process the packet (Task 3.4)
			// For now, just log occasionally to avoid flooding
			if packetCount%100 == 0 {
				log.Printf("Processed %d packets...\n", packetCount)
			}
			_ = packet // Use packet variable to avoid unused error
		}
	}
}
