package monitor

import (
	"fmt"
	"log"
	"network-monitor/internal/capture" // Assuming internal/capture is in the module path
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// Monitor holds the state and configuration for network monitoring.
type Monitor struct {
	interfaceName  string
	thresholdBytes float64 // Bytes per second
	interval       time.Duration
	handle         *pcap.Handle
	packetSource   *gopacket.PacketSource
}

// NewMonitor creates and initializes a new Monitor instance.
func NewMonitor(iface string, threshold float64, interval time.Duration) (*Monitor, error) {
	pktSource, handle, err := capture.StartCapture(iface)
	if err != nil {
		return nil, fmt.Errorf("could not start capture: %w", err)
	}

	m := &Monitor{
		interfaceName:  iface, // Store the actual interface name used
		thresholdBytes: threshold,
		interval:       interval,
		handle:         handle,
		packetSource:   pktSource,
	}
	// Assign the actually used interface name if one wasn't specified
	if iface == "" && handle != nil {
		// This relies on StartCapture logging the used interface,
		// a cleaner approach might be to have StartCapture return it.
		// For now, let's assume StartCapture correctly logged it or was given one.
		// If StartCapture modifies interfaceName passed to it, we need to handle that.
		// A better way: StartCapture could return the used interface name.
		// TODO: Modify StartCapture to return the used interface name.
		log.Printf("Monitoring on automatically selected interface. Check logs for name.")
	} else {
		m.interfaceName = iface
	}

	log.Printf("Monitor initialized. Interface: %s, Threshold: %.2f B/s, Interval: %s",
		m.interfaceName, m.thresholdBytes, m.interval)

	return m, nil
}

// Run starts the continuous monitoring process.
// It calculates traffic speed over the specified interval and logs a message
// if the speed exceeds the threshold.
func (m *Monitor) Run() {
	defer m.handle.Close() // Ensure the handle is closed when Run() exits

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	var bytesSinceLastTick int64
	intervalStartTime := time.Now()

	log.Printf("Starting monitoring loop...")

	for {
		select {
		case <-ticker.C:
			// Calculate speed for the completed interval
			elapsedTime := time.Since(intervalStartTime).Seconds()
			if elapsedTime == 0 {
				continue // Avoid division by zero if the interval is extremely short
			}

			speedBytesPerSec := float64(bytesSinceLastTick) / elapsedTime
			log.Printf("Interval Check: Duration=%.2fs, Bytes=%d, Speed=%.2f B/s",
				elapsedTime, bytesSinceLastTick, speedBytesPerSec)

			// Check against threshold
			if speedBytesPerSec > m.thresholdBytes {
				m.notifyThresholdExceeded(speedBytesPerSec)
			}

			// Reset for the next interval
			bytesSinceLastTick = 0
			intervalStartTime = time.Now()

		case packet, ok := <-m.packetSource.Packets():
			if !ok {
				log.Println("Packet source closed.")
				return // Exit if the packet source is closed
			}
			// Accumulate packet size
			bytesSinceLastTick += int64(len(packet.Data()))
		}
	}
}

// notifyThresholdExceeded is called when the traffic speed exceeds the threshold.
func (m *Monitor) notifyThresholdExceeded(currentSpeed float64) {
	// For now, just log a warning. This can be extended later.
	log.Printf("ALERT: Network speed threshold exceeded! Current: %.2f B/s, Threshold: %.2f B/s",
		currentSpeed, m.thresholdBytes)
}

// Close manually stops the capture and closes the handle.
func (m *Monitor) Close() {
	if m.handle != nil {
		log.Println("Closing pcap handle.")
		m.handle.Close()
		m.handle = nil // Prevent double closing
	}
}
