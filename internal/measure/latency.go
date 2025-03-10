package measure

import (
	"context"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/measure/latency"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/cespare/xxhash"
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

	h := xxhash.Sum64String(key)

	randSleep := time.Duration(float64(5*time.Second) * (float64(h) / (1 << 64)))
	time.Sleep(randSleep)

	test := &latency.Client{
		SourceIP: net.ParseIP(worker.sourceIP),
		TargetIP: net.ParseIP(worker.targetIP),
	}

	worker.logger.Debugf("Starting latency measurement on %s\n", worker.iface.Name)
	wg := &sync.WaitGroup{}
	workerCtx, workerCancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-ticker.C:
			ctx, _ := context.WithTimeout(workerCtx, 5*time.Second)

			wg.Add(1)
			go func() {
				defer wg.Done()
				// measure
				data, err := test.MeasureTCP(ctx, worker.targetPort)
				if err != nil {
					worker.logger.Errorf("error measuring TCP latency: %v", err)
				}

				// generate metrics
				generateLatencyMetrics(data, worker.iface)
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				data, err := test.MeasureICMP(ctx)
				if err != nil {
					worker.logger.Errorf("error measuring ICMP latency: %v", err)
				}

				// generate metrics
				generateLatencyMetrics(data, worker.iface)

			}()

		case <-worker.stopCh:
			worker.logger.Info("stopping latency measurement")

			// stop all measurements
			ticker.Stop()
			workerCancel()
			wg.Wait()

			// remove worker metrics
			err := unregisterLatencyMetrics(worker.iface)
			if err != nil {
				worker.logger.Errorf("error unregistering latency metrics: %v", err)
			}

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

func unregisterLatencyMetrics(iface netctl.WGInterface) error {
	pathName := generatePathName(iface.LocalID, iface.RemoteID)

	metrics := []*prometheus.GaugeVec{
		latencyStatus,
		latencyLoss,
		latencyDuration,
		latencyJitter,
	}

	for _, metric := range metrics {
		for _, protocol := range []string{"tcp", "icmp"} {
			ok := metric.DeleteLabelValues(
				protocol,
				iface.LocalID,
				iface.RemoteID,
				iface.IPVersion,
				pathName,
			)

			if !ok {
				return fmt.Errorf("failed to delete %s latency metrics for %s", protocol, iface.Name)
			}
		}
	}

	return nil
}
