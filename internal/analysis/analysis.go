package analysis

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type ConfigForAggregator struct {
	IntervalSeconds int
}

type TrafficData struct {
	Bytes int64
}

type Aggregator struct {
	mu           sync.RWMutex
	intervalData map[string]*TrafficData
	interval     time.Duration
	ticker       *time.Ticker
	stopChan     chan struct{}
	resultsChan  chan map[string]*TrafficData
	packetSource *gopacket.PacketSource
	log          *log.Logger
}

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
		resultsChan:  make(chan map[string]*TrafficData),
		packetSource: packetSource,
		log:          logger,
	}
	go agg.run()
	go agg.processPackets()
	return agg, agg.resultsChan
}

func (a *Aggregator) Stop() {
	close(a.stopChan)
	a.ticker.Stop()

}

func (a *Aggregator) processPackets() {
	for {
		select {
		case <-a.stopChan:
			a.log.Println("Stopping packet processing.")
			return
		case packet, ok := <-a.packetSource.Packets():
			if !ok {
				a.log.Println("Packet source channel closed.")

				close(a.stopChan)
				return
			}
			a.aggregatePacket(packet)
		}
	}
}

func (a *Aggregator) aggregatePacket(packet gopacket.Packet) {
	var srcIP net.IP
	var packetSize int

	ip4Layer := packet.Layer(layers.LayerTypeIPv4)
	if ip4Layer != nil {
		ip4, _ := ip4Layer.(*layers.IPv4)
		srcIP = ip4.SrcIP
		packetSize = len(ip4.Payload) + len(ip4.BaseLayer.Contents)
	} else {

		ip6Layer := packet.Layer(layers.LayerTypeIPv6)
		if ip6Layer != nil {
			ip6, _ := ip6Layer.(*layers.IPv6)
			srcIP = ip6.SrcIP
			packetSize = len(ip6.Payload) + len(ip6.BaseLayer.Contents)

			if packet.Metadata() != nil {
				packetSize = packet.Metadata().Length
			}

		}
	}

	if srcIP == nil || packetSize == 0 {
		return
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

func (a *Aggregator) run() {
	defer close(a.resultsChan)
	for {
		select {
		case <-a.ticker.C:
			a.processInterval()
		case <-a.stopChan:
			a.log.Println("Stopping aggregator ticker.")

			a.processInterval()
			return
		}
	}
}

func (a *Aggregator) processInterval() {
	a.mu.Lock()

	intervalSnapshot := make(map[string]*TrafficData, len(a.intervalData))
	totalBytes := int64(0)
	for ip, data := range a.intervalData {
		intervalSnapshot[ip] = &TrafficData{Bytes: data.Bytes}
		totalBytes += data.Bytes
	}

	a.intervalData = make(map[string]*TrafficData)
	a.mu.Unlock()

	intervalSeconds := float64(a.interval.Seconds())
	if intervalSeconds <= 0 {
		intervalSeconds = 1
	}

	overallSpeedMbps := (float64(totalBytes) * 8) / (intervalSeconds * 1_000_000)

	a.log.Printf("Interval finished. Total Bytes: %d, Overall Speed: %.2f Mbps\n", totalBytes, overallSpeedMbps)

	select {
	case a.resultsChan <- intervalSnapshot:

	case <-a.stopChan:

		a.log.Println("Aggregator stopping, discarding last interval result.")
	default:

		a.resultsChan <- intervalSnapshot

	}

}

func CalculateSpeedMbps(bytes int64, interval time.Duration) float64 {
	intervalSeconds := interval.Seconds()
	if intervalSeconds <= 0 {
		return 0.0
	}
	return (float64(bytes) * 8) / (intervalSeconds * 1_000_000)
}
