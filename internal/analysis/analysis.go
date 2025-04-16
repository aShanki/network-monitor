package analysis

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	// "network-monitor/internal/config" // Keep commented until Task 3.2
)

// ConfigForAggregator defines the config needed by the aggregator
// This avoids a direct dependency on the full config structure for now.
type ConfigForAggregator struct {
	IntervalSeconds int
}

// TrafficData holds the aggregated byte count for an IP address.
type TrafficData struct {
	Bytes int64
	// Add other metrics if needed later, e.g., packet count
}

// Aggregator collects traffic data per source IP over intervals.
type Aggregator struct {
	mu           sync.RWMutex
	intervalData map[string]*TrafficData // IP -> TrafficData for the current interval
	interval     time.Duration
	ticker       *time.Ticker
	stopChan     chan struct{}
	resultsChan  chan map[string]*TrafficData // Channel to send results for further processing
	packetSource *gopacket.PacketSource
	log          *log.Logger
}

// NewAggregator creates and starts a new Aggregator.
// Accepts ConfigForAggregator instead of the full config.Config
func NewAggregator(cfg *ConfigForAggregator, packetSource *gopacket.PacketSource, logger *log.Logger) (*Aggregator, chan map[string]*TrafficData) {
	if logger == nil {
		logger = log.Default()
	}
	if cfg.IntervalSeconds <= 0 {
		logger.Println("Warning: IntervalSeconds is zero or negative, defaulting to 5 seconds.")
		cfg.IntervalSeconds = 5
	}
	interval := time.Duration(cfg.IntervalSeconds) * time.Second
	agg := &Aggregator{
		intervalData: make(map[string]*TrafficData),
		interval:     interval,
		ticker:       time.NewTicker(interval),
		stopChan:     make(chan struct{}),
		resultsChan:  make(chan map[string]*TrafficData), // Unbuffered for now
		packetSource: packetSource,
		log:          logger,
	}
	go agg.run()
	go agg.processPackets()
	return agg, agg.resultsChan
}

// Stop halts the aggregator.
func (a *Aggregator) Stop() {
	close(a.stopChan)
	a.ticker.Stop()
	// Potentially signal packet processing goroutine to stop if needed
}

// processPackets reads packets and aggregates data.
func (a *Aggregator) processPackets() {
	for {
		select {
		case <-a.stopChan:
			a.log.Println("Stopping packet processing.")
			return
		case packet, ok := <-a.packetSource.Packets():
			if !ok {
				a.log.Println("Packet source channel closed.")
				// Optionally signal main routine or attempt recovery
				close(a.stopChan) // Ensure the ticker goroutine also stops
				return
			}
			a.aggregatePacket(packet)
		}
	}
}

// aggregatePacket extracts relevant information and updates interval data.
func (a *Aggregator) aggregatePacket(packet gopacket.Packet) {
	var srcIP net.IP
	var packetSize int

	// Handle IPv4
	ip4Layer := packet.Layer(layers.LayerTypeIPv4)
	if ip4Layer != nil {
		ip4, _ := ip4Layer.(*layers.IPv4)
		srcIP = ip4.SrcIP
		packetSize = len(ip4.Payload) + len(ip4.BaseLayer.Contents) // Use IP total length
	} else {
		// Handle IPv6
		ip6Layer := packet.Layer(layers.LayerTypeIPv6)
		if ip6Layer != nil {
			ip6, _ := ip6Layer.(*layers.IPv6)
			srcIP = ip6.SrcIP
			packetSize = len(ip6.Payload) + len(ip6.BaseLayer.Contents) // Use IPv6 payload length + header? Check gopacket docs. Often Length field is just payload. Using packet.Metadata().Length might be safer.
			// For simplicity, let's use metadata length if available
			if packet.Metadata() != nil {
				packetSize = packet.Metadata().Length
			}

		}
	}

	if srcIP == nil || packetSize == 0 {
		return // Not an IP packet we can analyze or empty
	}

	srcIPStr := srcIP.String()

	a.mu.Lock()
	defer a.mu.Unlock()

	data, exists := a.intervalData[srcIPStr]
	if !exists {
		data = &TrafficData{}
		a.intervalData[srcIPStr] = data
	}
	data.Bytes += int64(packetSize)
}

// run handles the ticker and processes data at each interval.
func (a *Aggregator) run() {
	defer close(a.resultsChan) // Close results channel when run loop exits
	for {
		select {
		case <-a.ticker.C:
			a.processInterval()
		case <-a.stopChan:
			a.log.Println("Stopping aggregator ticker.")
			// Process any remaining data before stopping?
			a.processInterval() // Process the last partial interval
			return
		}
	}
}

// processInterval calculates speed and sends data for the completed interval.
func (a *Aggregator) processInterval() {
	a.mu.Lock()
	// Deep copy the map to send, so the receiver doesn't race with the reset
	intervalSnapshot := make(map[string]*TrafficData, len(a.intervalData))
	totalBytes := int64(0)
	for ip, data := range a.intervalData {
		intervalSnapshot[ip] = &TrafficData{Bytes: data.Bytes} // Copy data
		totalBytes += data.Bytes
	}
	// Reset for the next interval *before* unlocking
	a.intervalData = make(map[string]*TrafficData)
	a.mu.Unlock() // Unlock before potentially blocking on channel send

	// Calculate overall speed for the interval
	// intervalSeconds := a.interval.Seconds() // Get interval duration correctly
	intervalSeconds := float64(a.interval.Seconds()) // Use float64 for calculation
	if intervalSeconds <= 0 {
		intervalSeconds = 1 // Avoid division by zero if interval is tiny or zero
	}

	// Speed in Mbps = (Total Bytes * 8 bits/byte) / (Interval Seconds * 1,000,000 bits/megabit)
	overallSpeedMbps := (float64(totalBytes) * 8) / (intervalSeconds * 1_000_000)

	a.log.Printf("Interval finished. Total Bytes: %d, Overall Speed: %.2f Mbps\n", totalBytes, overallSpeedMbps)

	// Send the snapshot for further processing (threshold check, top talkers)
	// This might block if the receiver isn't ready. Consider buffered channel or dropping data if necessary.
	select {
	case a.resultsChan <- intervalSnapshot:
		// Successfully sent
	case <-a.stopChan:
		// Aggregator stopping, don't block trying to send
		a.log.Println("Aggregator stopping, discarding last interval result.")
	default:
		// Receiver not ready (channel full or no receiver).
		// Decide strategy: block (current behavior with unbuffered), drop, or use buffered channel.
		// For now, let's log and drop if it would block indefinitely (though unbuffered will block).
		// A buffered channel might be better here.
		// Re-evaluate based on how resultsChan is consumed.
		// Sending on unbuffered channel:
		a.resultsChan <- intervalSnapshot

	}

	// --- Placeholder for Task 3.5 ---
	// Here, or in the goroutine consuming resultsChan:
	// 1. Check if overallSpeedMbps > threshold
	// 2. If yes, sort intervalSnapshot by Bytes (desc)
	// 3. Get top N
	// 4. Calculate individual speeds
	// 5. Send to Discord
	// --- End Placeholder ---

}

// CalculateSpeedMbps calculates speed in Mbps for given bytes over the interval duration.
// This can be used for overall speed or individual IP speeds.
func CalculateSpeedMbps(bytes int64, interval time.Duration) float64 {
	intervalSeconds := interval.Seconds()
	if intervalSeconds <= 0 {
		return 0.0
	}
	return (float64(bytes) * 8) / (intervalSeconds * 1_000_000)
}
