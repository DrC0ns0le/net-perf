package measure

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/measure/latency"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	latencyStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "network_latency_status",
		Help: "outcome of network_latency",
	}, []string{"type", "source", "target", "version", "path"})
	latencyDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "network_latency_duration",
		Help: "network_latency in microseconds",
	}, []string{"type", "source", "target", "version", "path"})
	latencyJitter = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "network_latency_jitter",
		Help: "network_latency jitter in microseconds",
	}, []string{"type", "source", "target", "version", "path"})
	latencyLoss = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "network_latency_loss",
		Help: "network_latency packet loss",
	}, []string{"type", "source", "target", "version", "path"})
)

func startLatencyWorker(worker *Worker) {
	key := fmt.Sprintf("iface=%s, sourceIP=%s, targetIP=%s, targetPort=%d",
		worker.iface.Name, worker.sourceIP, worker.targetIP, worker.targetPort)

	hash := sha256.Sum256([]byte(key))
	h := binary.BigEndian.Uint64(hash[:8])

	randSleep := time.Duration(float64(5*time.Second) * (float64(h) / (1 << 64)))
	time.Sleep(randSleep)

	// log.Printf("Starting latency measurement on %s\n", worker.iface.Name)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			go func() {
				// measure
				data, err := latency.MeasureTCP(ctx, worker.sourceIP, worker.targetIP, worker.targetPort)
				if err != nil {
					log.Println("%s: error measuring TCP latency:", worker.iface.Name, err)
				}

				// generate metrics
				generateLatencyMetrics(data, worker.iface)
			}()

			go func() {
				data, err := latency.MeasureICMP(ctx, worker.sourceIP, worker.targetIP)
				if err != nil {
					log.Println("Error measuring ICMP latency:", err)
				}

				// generate metrics
				generateLatencyMetrics(data, worker.iface)

			}()

		case <-worker.stopCh:
			log.Printf("Stopping latency measurement on %s\n", worker.iface.Name)

			// Set metrics to NaN
			generateLatencyMetrics(latency.Result{
				Protocol: "tcp",
				Status:   0,
			}, worker.iface)
			generateLatencyMetrics(latency.Result{
				Protocol: "icmp",
				Status:   0,
			}, worker.iface)

			return
		}
	}
}

func generateLatencyMetrics(data latency.Result, iface netctl.WGInterface) {
	var avgLatency float64
	var jitter float64
	var loss float64
	if data.Status == 1 {
		avgLatency = float64(data.AvgLatency)
		jitter = float64(data.Jitter)
		loss = float64(data.Loss)
	} else {
		avgLatency = math.NaN()
		jitter = math.NaN()
		loss = math.NaN()
	}

	pathName := generatePathName(iface.LocalID, iface.RemoteID)

	latencyStatus.WithLabelValues(
		data.Protocol,
		iface.LocalID,
		iface.RemoteID,
		iface.IPVersion,
		pathName,
	).Set(float64(data.Status))

	latencyLoss.WithLabelValues(
		data.Protocol,
		iface.LocalID,
		iface.RemoteID,
		iface.IPVersion,
		pathName,
	).Set(loss)

	latencyDuration.WithLabelValues(
		data.Protocol,
		iface.LocalID,
		iface.RemoteID,
		iface.IPVersion,
		pathName,
	).Set(avgLatency)

	latencyJitter.WithLabelValues(
		data.Protocol,
		iface.LocalID,
		iface.RemoteID,
		iface.IPVersion,
		pathName,
	).Set(jitter)
}
