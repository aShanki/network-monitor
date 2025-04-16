package metrics

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	networkSpeed = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "network_speed_mbps",
			Help: "Current network speed in Mbps",
		},
		[]string{"interface", "direction"},
	)

	networkTraffic = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "network_traffic_bytes_total",
			Help: "Total network traffic in bytes",
		},
		[]string{"interface", "direction"},
	)

	topTalkers = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "network_top_talkers_mbps",
			Help: "Top network talkers by speed in Mbps",
		},
		[]string{"interface", "ip_address"},
	)

	thresholdExceeded = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "network_threshold_exceeded",
			Help: "Whether the network speed threshold is exceeded (1 for yes, 0 for no)",
		},
	)
)

type MetricsServer struct {
	server *http.Server
}

func NewMetricsServer(port string) *MetricsServer {
	if port == "" {
		port = "9090"
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return &MetricsServer{
		server: server,
	}
}

func (m *MetricsServer) Start() {
	go func() {
		log.Printf("Starting Prometheus metrics server on %s", m.server.Addr)
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()
}

func (m *MetricsServer) Stop() {
	log.Println("Stopping metrics server...")
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	if err := m.server.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down metrics server: %v", err)
	}
}

func contextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

func UpdateNetworkSpeed(interfaceName string, speedMbps float64) {
	networkSpeed.WithLabelValues(interfaceName, "total").Set(speedMbps)
}

func UpdateNetworkTraffic(interfaceName string, bytes int64) {
	networkTraffic.WithLabelValues(interfaceName, "total").Add(float64(bytes))
}

func UpdateTopTalkers(interfaceName string, ipSpeeds map[string]float64) {

	topTalkers.Reset()

	for ip, speed := range ipSpeeds {
		topTalkers.WithLabelValues(interfaceName, ip).Set(speed)
	}
}

func UpdateThresholdStatus(exceeded bool) {
	if exceeded {
		thresholdExceeded.Set(1)
	} else {
		thresholdExceeded.Set(0)
	}
}
