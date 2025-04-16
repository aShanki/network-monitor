package monitor

import (
	"fmt"
	"log"
	"network-monitor/internal/analysis"
	"network-monitor/internal/capture"
	"network-monitor/internal/config"
	"network-monitor/internal/discord"
	"network-monitor/internal/metrics"
	"sort"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

type Monitor struct {
	cfg           *config.Config
	interfaceName string
	handle        *pcap.Handle
	packetSource  *gopacket.PacketSource
	aggregator    *analysis.Aggregator
	resultsChan   <-chan map[string]*analysis.TrafficData
	stopChan      chan struct{}
	metricsServer *metrics.MetricsServer
}

func NewMonitor(cfg *config.Config) (*Monitor, error) {
	pktSource, handle, err := capture.StartCapture(cfg.InterfaceName)
	if err != nil {
		return nil, fmt.Errorf("could not start capture: %w", err)
	}

	aggCfg := &analysis.ConfigForAggregator{IntervalSeconds: cfg.IntervalSeconds}
	agg, resultsChan := analysis.NewAggregator(aggCfg, pktSource, log.Default())

	m := &Monitor{
		cfg:           cfg,
		interfaceName: cfg.InterfaceName,
		handle:        handle,
		packetSource:  pktSource,
		aggregator:    agg,
		resultsChan:   resultsChan,
		stopChan:      make(chan struct{}),
	}

	if cfg.InterfaceName == "" && handle != nil {
		log.Printf("Monitoring on automatically selected interface. Check logs for name.")
		m.interfaceName = "Auto-Selected"
	} else {
		m.interfaceName = cfg.InterfaceName
	}

	log.Printf("Monitor initialized. Interface: %s, Threshold: %.2f Mbps, Interval: %ds, TopN: %d",
		m.interfaceName, m.cfg.ThresholdMbps, m.cfg.IntervalSeconds, m.cfg.TopN)

	// Initialize Prometheus metrics server if enabled
	if cfg.MetricsEnabled {
		m.metricsServer = metrics.NewMetricsServer(cfg.MetricsPort)
		m.metricsServer.Start()
		log.Printf("Prometheus metrics endpoint initialized on port %s", cfg.MetricsPort)
	}

	go func() {
		err := discord.SendInitNotification(m.cfg.WebhookURL, m.interfaceName, m.cfg.ThresholdMbps, m.cfg.IntervalSeconds)
		if err != nil {
			log.Printf("Error sending Discord init notification: %v", err)
		}
	}()

	return m, nil
}

func (m *Monitor) Run() {
	log.Printf("Starting monitoring loop...")

	for {
		select {
		case intervalData, ok := <-m.resultsChan:
			if !ok {
				log.Println("Aggregator results channel closed. Monitor stopping.")
				return
			}

			m.processIntervalData(intervalData)

		case <-m.stopChan:
			log.Println("Monitor stopping loop.")
			return
		}
	}
}

func (m *Monitor) processIntervalData(intervalData map[string]*analysis.TrafficData) {
	interval := m.cfg.GetIntervalDuration()
	overallBytes := int64(0)
	ipSpeeds := make(map[string]float64)

	for ip, data := range intervalData {
		overallBytes += data.Bytes
		ipSpeedMbps := analysis.CalculateSpeedMbps(data.Bytes, interval)
		ipSpeeds[ip] = ipSpeedMbps
	}

	overallSpeedMbps := analysis.CalculateSpeedMbps(overallBytes, interval)

	log.Printf("Interval Check: Duration=%.2fs, Total Bytes=%d, Overall Speed=%.2f Mbps",
		interval.Seconds(), overallBytes, overallSpeedMbps)

	// Update Prometheus metrics if enabled
	if m.cfg.MetricsEnabled {
		metrics.UpdateNetworkSpeed(m.interfaceName, overallSpeedMbps)
		metrics.UpdateNetworkTraffic(m.interfaceName, overallBytes)
		metrics.UpdateTopTalkers(m.interfaceName, ipSpeeds)

		// Set threshold exceeded status
		thresholdExceeded := overallSpeedMbps > m.cfg.ThresholdMbps
		metrics.UpdateThresholdStatus(thresholdExceeded)
	}

	if overallSpeedMbps > m.cfg.ThresholdMbps {
		m.notifyThresholdExceeded(overallSpeedMbps, ipSpeeds)
	}
}

func (m *Monitor) notifyThresholdExceeded(currentSpeedMbps float64, ipSpeeds map[string]float64) {
	log.Printf("ALERT: Network speed threshold exceeded! Current: %.2f Mbps, Threshold: %.2f Mbps",
		currentSpeedMbps, m.cfg.ThresholdMbps)

	if m.cfg.WebhookURL == "" {
		return
	}

	type ipSpeedPair struct {
		IP    string
		Speed float64
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

	go func() {
		err := discord.SendDiscordNotification(m.cfg.WebhookURL, topTalkersMap, m.cfg.ThresholdMbps, m.cfg.IntervalSeconds)
		if err != nil {
			log.Printf("Error sending Discord threshold notification: %v", err)
		}
	}()
}

func (m *Monitor) Close() {
	log.Println("Monitor Close requested.")

	close(m.stopChan)

	if m.aggregator != nil {
		m.aggregator.Stop()
	}

	// Stop metrics server if it was started
	if m.metricsServer != nil {
		m.metricsServer.Stop()
	}

	log.Println("Monitor closed.")
}
