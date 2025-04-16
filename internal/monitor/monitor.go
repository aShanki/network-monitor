package monitor

import (
	"fmt"
	"log"
	"network-monitor/internal/analysis"
	"network-monitor/internal/capture"
	"network-monitor/internal/config"  // Need full config
	"network-monitor/internal/discord" // Need discord functions
	"sort"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// Monitor holds the state and configuration for network monitoring.
type Monitor struct {
	cfg           *config.Config // Store the full config
	interfaceName string
	handle        *pcap.Handle
	packetSource  *gopacket.PacketSource
	aggregator    *analysis.Aggregator
	resultsChan   <-chan map[string]*analysis.TrafficData
	stopChan      chan struct{}
}

// NewMonitor creates and initializes a new Monitor instance.
func NewMonitor(cfg *config.Config) (*Monitor, error) {
	pktSource, handle, err := capture.StartCapture(cfg.InterfaceName)
	if err != nil {
		return nil, fmt.Errorf("could not start capture: %w", err)
	}

	// Create the aggregator
	aggCfg := &analysis.ConfigForAggregator{IntervalSeconds: cfg.IntervalSeconds}
	agg, resultsChan := analysis.NewAggregator(aggCfg, pktSource, log.Default())

	m := &Monitor{
		cfg:           cfg,
		interfaceName: cfg.InterfaceName, // Store the potentially auto-selected name later
		handle:        handle,
		packetSource:  pktSource,
		aggregator:    agg,
		resultsChan:   resultsChan,
		stopChan:      make(chan struct{}),
	}

	// Assign the actually used interface name if one wasn't specified
	if cfg.InterfaceName == "" && handle != nil {
		// TODO: Modify StartCapture to return the used interface name.
		// For now, assume it was logged, and we'll use "Auto-Selected" for notifications.
		log.Printf("Monitoring on automatically selected interface. Check logs for name.")
		m.interfaceName = "Auto-Selected" // Use a placeholder for display
	} else {
		m.interfaceName = cfg.InterfaceName // Use the one from config
	}

	log.Printf("Monitor initialized. Interface: %s, Threshold: %.2f Mbps, Interval: %ds, TopN: %d",
		m.interfaceName, m.cfg.ThresholdMbps, m.cfg.IntervalSeconds, m.cfg.TopN)

	// Send initialization notification
	go func() {
		err := discord.SendInitNotification(m.cfg.WebhookURL, m.interfaceName, m.cfg.ThresholdMbps, m.cfg.IntervalSeconds)
		if err != nil {
			log.Printf("Error sending Discord init notification: %v", err)
		}
	}()

	return m, nil
}

// Run starts the continuous monitoring process.
// It consumes results from the aggregator and sends notifications immediately
// if the threshold is exceeded.
func (m *Monitor) Run() {
	log.Printf("Starting monitoring loop...")

	for {
		select {
		case intervalData, ok := <-m.resultsChan:
			if !ok {
				log.Println("Aggregator results channel closed. Monitor stopping.")
				return
			}
			// Process the aggregated data for the interval
			m.processIntervalData(intervalData)

		case <-m.stopChan:
			log.Println("Monitor stopping loop.")
			return
		}
	}
}

// processIntervalData calculates speeds, checks threshold, and sends notifications.
func (m *Monitor) processIntervalData(intervalData map[string]*analysis.TrafficData) {
	interval := m.cfg.GetIntervalDuration()
	overallBytes := int64(0)
	ipSpeeds := make(map[string]float64) // IP -> Speed (Mbps)

	for ip, data := range intervalData {
		overallBytes += data.Bytes
		ipSpeedMbps := analysis.CalculateSpeedMbps(data.Bytes, interval)
		ipSpeeds[ip] = ipSpeedMbps
	}

	overallSpeedMbps := analysis.CalculateSpeedMbps(overallBytes, interval)

	log.Printf("Interval Check: Duration=%.2fs, Total Bytes=%d, Overall Speed=%.2f Mbps",
		interval.Seconds(), overallBytes, overallSpeedMbps)

	// Check against threshold
	if overallSpeedMbps > m.cfg.ThresholdMbps {
		m.notifyThresholdExceeded(overallSpeedMbps, ipSpeeds)
	}
}

// notifyThresholdExceeded logs the alert and sends a Discord notification.
func (m *Monitor) notifyThresholdExceeded(currentSpeedMbps float64, ipSpeeds map[string]float64) {
	log.Printf("ALERT: Network speed threshold exceeded! Current: %.2f Mbps, Threshold: %.2f Mbps",
		currentSpeedMbps, m.cfg.ThresholdMbps)

	if m.cfg.WebhookURL == "" {
		return // Don't attempt notification if URL is not set
	}

	// Prepare top talkers data
	type ipSpeedPair struct {
		IP    string
		Speed float64 // Mbps
	}
	var sortedTalkers []ipSpeedPair
	for ip, speed := range ipSpeeds {
		sortedTalkers = append(sortedTalkers, ipSpeedPair{IP: ip, Speed: speed})
	}
	sort.Slice(sortedTalkers, func(i, j int) bool {
		return sortedTalkers[i].Speed > sortedTalkers[j].Speed
	})

	topN := m.cfg.TopN
	if len(sortedTalkers) < topN {
		topN = len(sortedTalkers)
	}

	topTalkersMap := make(map[string]float64)
	for i := 0; i < topN; i++ {
		topTalkersMap[sortedTalkers[i].IP] = sortedTalkers[i].Speed
	}

	// Send notification in a separate goroutine to avoid blocking the monitor loop
	go func() {
		err := discord.SendDiscordNotification(m.cfg.WebhookURL, topTalkersMap, m.cfg.ThresholdMbps, m.cfg.IntervalSeconds)
		if err != nil {
			log.Printf("Error sending Discord threshold notification: %v", err)
		}
	}()
}

// Close manually stops the capture and closes the handle.
func (m *Monitor) Close() {
	log.Println("Monitor Close requested.")
	// Signal the run loop to stop
	close(m.stopChan)

	// Stop the aggregator (which will close the results channel)
	if m.aggregator != nil {
		m.aggregator.Stop()
	}

	// The packet source is owned by the aggregator now, no need to close handle here
	// if m.handle != nil {
	// 	log.Println("Closing pcap handle.")
	// 	m.handle.Close()
	// 	m.handle = nil // Prevent double closing
	// }
	log.Println("Monitor closed.")
}
