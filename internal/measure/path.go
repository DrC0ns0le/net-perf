package measure

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/measure/pathping"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/cespare/xxhash"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pathLatencyStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "network_path_latency_status",
		Help: "outcome of network_path_latency",
	}, []string{"type", "source", "target", "path"})
	pathLatencyDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "network_path_latency_duration",
		Help: "network_path_latency in microseconds",
	}, []string{"type", "source", "target", "path"})
	pathLatencyLoss = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "network_path_latency_loss",
		Help: "network_path_latency packet loss",
	}, []string{"type", "source", "target", "path"})
)

func startPathLatencyWorker(worker *Worker) {
	key := fmt.Sprintf("sourceIP=%s, targetIP=%s",
		worker.sourceIP, worker.targetIP)

	h := xxhash.Sum64String(key)

	randSleep := time.Duration(float64(5*time.Second) * (float64(h) / (1 << 64)))
	time.Sleep(randSleep)

	worker.logger.Debugf("Starting path latency measurement for %s\n", worker.iface.RemoteID)
	wg := &sync.WaitGroup{}
	ticker := time.NewTicker(15 * time.Second)

	ppClient := pathping.NewClient(10, 250*time.Millisecond)
	path := []int{worker.iface.LocalIDInt, worker.iface.RemoteIDInt, worker.iface.LocalIDInt}
	for {
		select {
		case <-ticker.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				data, err := ppClient.Measure(path)
				if err != nil {
					worker.logger.Errorf("error measuring path latency: %v", err)
				}

				generatePathLatencyMetrics(data, worker.iface)
			}()

		case <-worker.stopCh:
			worker.logger.Info("stopping latency measurement")

			// stop all measurements
			ticker.Stop()
			wg.Wait()

			// remove worker metrics
			err := unregisterPathLatencyMetrics(worker.iface)
			if err != nil {
				worker.logger.Errorf("error unregistering latency metrics: %v", err)
			}

			return
		}
	}
}

func generatePathLatencyMetrics(data pathping.Result, iface netctl.WGInterface) {
	var avgLatency float64
	var loss float64
	if data.Status == 1 {
		avgLatency = float64(data.Duration.Microseconds())
		loss = data.Loss
	} else {
		avgLatency = math.NaN()
		loss = math.NaN()
	}

	pathName := generatePathName(iface.LocalID, iface.RemoteID)

	pathLatencyStatus.WithLabelValues(
		data.Protocol,
		iface.LocalID,
		iface.RemoteID,
		pathName,
	).Set(float64(data.Status))

	pathLatencyLoss.WithLabelValues(
		data.Protocol,
		iface.LocalID,
		iface.RemoteID,
		pathName,
	).Set(loss)

	pathLatencyDuration.WithLabelValues(
		data.Protocol,
		iface.LocalID,
		iface.RemoteID,
		pathName,
	).Set(avgLatency)
}

func unregisterPathLatencyMetrics(iface netctl.WGInterface) error {
	pathName := generatePathName(iface.LocalID, iface.RemoteID)

	metrics := []*prometheus.GaugeVec{
		pathLatencyDuration,
		pathLatencyLoss,
		pathLatencyStatus,
	}

	for _, metric := range metrics {
		for _, protocol := range []string{"pathping"} {
			ok := metric.DeleteLabelValues(
				protocol,
				iface.LocalID,
				iface.RemoteID,
				pathName,
			)

			if !ok {
				return fmt.Errorf("failed to delete %s latency metrics for %s", protocol, iface.Name)
			}
		}
	}

	return nil
}
